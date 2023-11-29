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
		io.WriteString(w, http.MethodGet)
	case http.MethodPost:
		// Parse the content type header
		mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
		if err != nil {
			errMsg := fmt.Sprintf("error understanding content-type: %v", err)
			ReturnHTTPErrorResponse(w, errMsg, http.StatusInternalServerError)
			return
		}

		// This in theory never happens
		if mediaType == "" {
			log.Println("media type not set")
			ReturnHTTPErrorResponse(w, "no content-type supplied", http.StatusBadRequest)
			return
		}
		/*
			multipart/form-data records contain binary (files) as well as alpahnumeric (payload) data

			This block handles parsing of the form submission and file storage

			An example cURL request is as follows
			curl -v -F "url=https://www.imagination.com" -F "file=@file.mp4" -X POST http://localhost:8080/api/rq/http

		*/
		if mediaType == "multipart/form-data" {
			//TODO: Set maxMemory in config or env var?
			if err := req.ParseMultipartForm(100000); err != nil {
				errMsg := fmt.Sprintf("error parsing multipart formdata, %v", err)
				ReturnHTTPErrorResponse(w, errMsg, http.StatusInternalServerError)
				return
			}

			// Get the file submitted as part of the request

			file, headers, err := req.FormFile("file")
			if err != nil {
				ReturnHTTPErrorResponse(w, "No form field with key 'file'", http.StatusBadRequest)
				return
			}

			// Store the file on disk
			srcFileName := headers.Filename
			dstFileName := record.Id
			if err := StoreFile(dstFileName, srcFileName, file); err != nil {
				var errMsg string
				var responseCode int
				switch {
				case errors.Is(err, FileExtError):
					errMsg = fmt.Sprintf("Failed to save file: %v", err.Error())
					responseCode = http.StatusBadRequest
				default:
					errMsg = fmt.Sprintf("Server errror saving file")
					responseCode = http.StatusInternalServerError
				}

				ReturnHTTPErrorResponse(w, errMsg, responseCode)
				return
			}

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
			/*

				Map the url.Values into a payload, to be marshalled into JSON.

				Where multiple values are sent for the same key, the library adds them all to a slice.
				We could store this as a plain string, but a JSON object makes sense right now.

				> curl -v -d "url=https://img.com&likes=stuff&likes=things" -X POST http://localhost:8080/api/rq/http
				< {"likes":["stuff","things"],"url":["https://img.com"]}
			*/

			payload := map[string][]string{}
			for x, y := range req.Form {
				payload[x] = y
			}

			out, _ := json.Marshal(payload)
			record.Payload = string(out)

		}

		//Get URL from Form and set it on RqRecord
		url := req.Form.Get("url")
		if url == "" {
			errMsg := fmt.Sprintf("no URL provided")
			ReturnHTTPErrorResponse(w, errMsg, http.StatusBadRequest)
		}
		record.Url = url

		// Set Content-Type on Record

		record.ContentType = mediaType

		// Store in DB
		record.Save()
		//	errMsg := fmt.Sprintf("error saving to database ", err)
		//	ReturnHTTPErrorResponse(w, errMsg, http.StatusInternalServerError)
		//	return
		//}

		// Return representation

		rqreq := RqRequest{
			Id:     getRqId(req),
			Record: record,
		}

		out, _ := json.Marshal(rqreq)
		io.WriteString(w, string(out))
	case http.MethodPatch:
		io.WriteString(w, http.MethodPatch)
	default:
		errMsg := fmt.Sprintf("Method %v not currently supported.", req.Method)
		ReturnHTTPErrorResponse(w, errMsg, http.StatusNotImplemented)
	}
	return
}

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
