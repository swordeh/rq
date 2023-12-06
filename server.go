package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"mime"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type RqRequest struct {
	Id     string   `json:"id"`
	Record RqRecord `json:"record"`
}

func RqHttpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		rqid := GenerateRequestId()
		ctx := req.Context()
		ctx = context.WithValue(ctx, "rqid", rqid)
		req = req.WithContext(ctx)
		log.Printf("processing request %v", rqid)

		next.ServeHTTP(w, req)
	})
}

// ReturnHTTPError returns an ErrorResponse back to the client if a request has failed.
func ReturnHTTPErrorResponse(w http.ResponseWriter, errorMessage string, status int) {
	w.WriteHeader(status)
	output, _ := json.Marshal(ErrorResponse{Error: errorMessage})
	io.WriteString(w, string(output))
}

func GenerateRequestId() string {
	return uuid.New().String()
}

//func HttpOnlyHandler(w http.ResponseWriter, req *http.Request) {
//	req.Context()
//	rqid := GenerateRequestId()
//	rq := RqRequest{
//		RqId: rqid,
//	}
//}

func QueueHttpHandler(w http.ResponseWriter, req *http.Request) {

	// Build up an RqRecord based on what we know so far.

	record := RqRecord{
		Id:     getRqId(req),
		Method: req.Method,
		Error:  "",
	}

	/*
		Content-Type can vary, depending on the type of POST.

		If binary data is being sent, it's likely being sent as multipart/form-data,
		for size optimisation, whereas alphanumeric data will typically be sent as
		application/x-www-form-urlencoded.

		Because a GET request does not send this header, it's parsed for appropriate
		methods only
	*/
	contentTypeHeader := req.Header.Get("Content-Type")

	log.Printf("Got a %v request", req.Method)

	switch req.Method {
	case http.MethodGet:
		querystring := req.URL.Query()
		url := querystring.Get("url")

		if url == "" {
			errMsg := fmt.Sprintf("no url supplied")
			ReturnHTTPErrorResponse(w, errMsg, http.StatusBadRequest)
			return
		}

		record.Url = url
		out, _ := json.Marshal(querystring)

		record.Payload = out
		record.Save()

		rqreq := RqRequest{
			Id:     getRqId(req),
			Record: record,
		}

		out, _ = json.Marshal(rqreq)
		io.WriteString(w, string(out))
		return

	case http.MethodPost:

		/*
			Headers

			Headers are passed to the onwards API from both the Request Headers

			The following request headers will automatically be removed to the request:
				* Content-Length
				* Accept
				* User-Agent
				* Content-Length (Stored as a separate field)

			Additional headers will automatically be passed on, unless excluded in config.

		*/

		mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
		if err != nil {
			errMsg := fmt.Sprintf("error parsing Content-Type: %v", err)
			ReturnHTTPErrorResponse(w, errMsg, http.StatusInternalServerError)
			return
		}

		// Check if content-type sent matches list of allowed types and reject if not
		if contains(config.Server.AllowedContentTypes, mediaType) == false {
			ReturnHTTPErrorResponse(w, "No or unsupported Content-Type supplied", http.StatusBadRequest)
			return
		}

		// This in theory never happens
		if mediaType == "" {
			log.Println("media type not set")
			ReturnHTTPErrorResponse(w, "no content-type supplied", http.StatusBadRequest)
			return
		}

		headers := map[string][]string{}
		serverExcludedHeaders := []string{"Content-Length", "User-Agent", "Content-Type", "Accept"}

		for _, key := range serverExcludedHeaders {
			if contains(config.Server.ExcludedHeaders, key) == false {
				config.Server.ExcludedHeaders = append(config.Server.ExcludedHeaders, key)
			}
		}

		//// Add default excluded headers if config does not already contain them
		//if len(config.Server.ExcludedHeaders) == 0 {
		//	config.Server.ExcludedHeaders = append(config.Server.ExcludedHeaders, serverExcludedHeaders)
		//}

		for key, values := range req.Header {
			if contains(config.Server.ExcludedHeaders, key) == false {
				headers[key] = values
			}
		}

		out, _ := json.Marshal(headers)
		record.Headers = out

		/*
			multipart/form-data records contain binary (files) as well as alpahnumeric (payload) data

			This block handles parsing of the form submission and file storage

			An example cURL request is as follows
			curl -v -F "url=https://www.imagination.com" -F "file=@file.mp4" -X POST http://localhost:8080/api/rq/http

		*/
		if mediaType == "multipart/form-data" {
			//TODO: Set maxMemory in config or env var?
			if err := req.ParseMultipartForm(320000); err != nil {
				errMsg := fmt.Sprintf("error parsing multipart formdata, %v", err)
				ReturnHTTPErrorResponse(w, errMsg, http.StatusInternalServerError)
				return
			}

			/*
				File Handler

				This block handles the file(s) uploaded to RQ, storing to disk and recording the
				file key used, to be passed to the onwards API.

				As req.FormFile() requires a key, RQ takes an opinionated approach to the keys provided,
				and puts the onus on the calling service to ensure keys match the onward API requirements.

				As such, a list of file keys in the request is stored and appended to the stored file name.
			*/

			if len(req.MultipartForm.File) == 0 {
				errMsg := fmt.Sprintf("no file submitted but Content-Type %v used", mediaType)
				ReturnHTTPErrorResponse(w, errMsg, http.StatusBadRequest)
				return
			}

			fileKeys := []string{}

			for key, _ := range req.MultipartForm.File {
				// Store the file on disk
				file, fileHeaders, err := req.FormFile(key)
				if err != nil {
					errMsg := fmt.Sprintf("server error getting file for key: %v, %v", key, err)
					ReturnHTTPErrorResponse(w, errMsg, 500)
					return
				}
				srcFileName := fileHeaders.Filename
				dstFileName := fmt.Sprintf("%v-%v", record.Id, key)
				if err := StoreFile(dstFileName, srcFileName, file); err != nil {
					var errMsg string
					var responseCode int
					switch {
					case errors.Is(err, FileExtError):
						errMsg = fmt.Sprintf("Check file extension: %v", err.Error())
						responseCode = http.StatusBadRequest
					default:
						errMsg = fmt.Sprintf("Server errror saving file")
						responseCode = http.StatusInternalServerError
					}

					ReturnHTTPErrorResponse(w, errMsg, responseCode)
					return
				}
				// TODO: Record key names into db
				fileKeys = append(fileKeys, key)

			}
			out, _ := json.Marshal(fileKeys)
			record.FileKeys = string(out)
		}

		/*
			application/x-www-form-urlencoded records are alphanumeric

			curl -v -d "url=https://imagination.com" -X POST http://localhost:8080/api/rq/http
		*/
		if mediaType == "application/x-www-form-urlencoded" {
			//TODO: Set maxMemory in config or env var?
			if err := req.ParseForm(); err != nil {
				errMsg := fmt.Sprintf("error parsing  formdata, %v", err)
				ReturnHTTPErrorResponse(w, errMsg, http.StatusInternalServerError)
				return
			}

		}

		url := req.Form.Get("url")
		if url == "" {
			errMsg := fmt.Sprintf("no URL provided")
			ReturnHTTPErrorResponse(w, errMsg, http.StatusBadRequest)
			return
		}
		record.Url = url

		/*

			Map the url.Values into a payload, to be marshalled into JSON.

			Where multiple values are sent for the same key, the library adds them all to a slice.
			We could store this as a plain string, but a JSON object makes sense right now.

			> curl -v -d "url=https://img.com&likes=stuff&likes=things" -X POST http://localhost:8080/api/rq/http
			< {"likes":["stuff","things"],"url":["https://img.com"]}
		*/
		payload := map[string][]string{}
		for key, values := range req.Form {
			payload[key] = values
		}

		// remove URL from stored payload, as this isn't sent onwards
		delete(payload, "url")

		out, _ = json.Marshal(payload)
		record.Payload = out

		// Set Content-Type on Record
		record.ContentType = mediaType

		// Store in DB
		record.Save()

		// Return representation
		rqreq := RqRequest{
			Id:     getRqId(req),
			Record: record,
		}

		out, _ = json.Marshal(rqreq)
		io.WriteString(w, string(out))

		return

	case http.MethodPatch:
		io.WriteString(w, http.MethodPatch)
	default:
		errMsg := fmt.Sprintf("Method %v not currently supported.", req.Method)
		ReturnHTTPErrorResponse(w, errMsg, http.StatusNotImplemented)
	}
	return
}

// contains iterates through a slice to check for the presence of str
func contains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// getRqId returns the RQ ID from the Request's Context
func getRqId(req *http.Request) string {
	rqidCtx := req.Context().Value("rqid")
	rqid := fmt.Sprint(rqidCtx)

	return rqid
}

func main() {
	// TODO: Move profile selection to CLI arg / env var
	if err := LoadConfigFile("default"); err != nil {
		log.Fatal("Could not load config file", err)
	}

	if err := OpenDatabase(); err != nil {
		log.Fatal("error opening database connection: ", err)
	}

	mux := http.NewServeMux()

	HttpRequestHandler := http.HandlerFunc(QueueHttpHandler)

	mux.Handle("/api/rq/http", RqHttpMiddleware(HttpRequestHandler))

	//http.HandleFunc("/api/rq/request", QueueMediaHandler)
	//htto.HandleFunc("/	api/rq/http", HttpOnlyHandler)
	log.Fatal(http.ListenAndServe(":8080", mux))

}
