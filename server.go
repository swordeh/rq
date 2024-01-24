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
	"rq/config"
	"rq/files"
	"rq/helpers"
	"rq/records"
)

type HttpError interface {
	error
	Status() int
}

type StatusError struct {
	StatusCode int
	Err        error
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type RqRequest struct {
	Id     string            `json:"id"`
	Record *records.RqRecord `json:"record"`
}

type RecordServer struct {
	Store     records.RecordStore
	FileStore files.FileStore
}

func (s StatusError) Error() string {
	return s.Err.Error()
}

func (s StatusError) Status() int {
	return s.StatusCode
}

func (rs *RecordServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	rqId := getRqId(req)
	record := records.RqRecord{
		Id:     rqId,
		Method: req.Method,
		Error:  "",
	}

	rqreq := RqRequest{
		Id:     rqId,
		Record: &record,
	}

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

		err := rs.saveRecord(record)
		if err != nil {
			switch e := err.(type) {
			case HttpError:
				ReturnHTTPErrorResponse(w, e.Error(), e.Status())
				return
			default:
				ReturnHTTPErrorResponse(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}

	case http.MethodPost:
		err := rs.HandleFormMethod(req, &record)
		if err != nil {
			switch e := err.(type) {
			case HttpError:
				ReturnHTTPErrorResponse(w, err.Error(), e.Status())
				return
			default:
				ReturnHTTPErrorResponse(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	case http.MethodPatch:
		err := rs.HandleFormMethod(req, &record)
		if err != nil {
			switch e := err.(type) {
			case HttpError:
				ReturnHTTPErrorResponse(w, err.Error(), e.Status())
				return
			default:
				ReturnHTTPErrorResponse(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	case http.MethodPut:
		err := rs.HandleFormMethod(req, &record)
		if err != nil {
			switch e := err.(type) {
			case HttpError:
				ReturnHTTPErrorResponse(w, err.Error(), e.Status())
				return
			default:
				ReturnHTTPErrorResponse(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

	default:
		ReturnHTTPErrorResponse(w, http.StatusText(http.StatusNotImplemented), http.StatusNotImplemented)
		return
	}

	out, _ := json.Marshal(rqreq)
	io.WriteString(w, string(out))

	return

}

func (rs *RecordServer) HandleFormMethod(req *http.Request, record *records.RqRecord) error {

	// Get RqId
	rqId := getRqId(req)

	// Get the Content-Type full header
	contentTypeHeader := req.Header.Get("Content-Type")
	log.Printf("%v: Got a %v request", rqId, req.Method)

	// Get the Content-Type parsed value
	mediaType, _, err := mime.ParseMediaType(contentTypeHeader)
	if err != nil {
		errMsg := fmt.Sprintf("error parsing Content-Type: %v", err)
		//ReturnHTTPErrorResponse(w, errMsg, http.StatusInternalServerError)
		return StatusError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New(errMsg),
		}

	}

	// Check that the content-type is supported
	if err = validateContentType(mediaType); err != nil {
		return StatusError{
			StatusCode: http.StatusBadRequest,
			Err:        err,
		}
	}

	// Content-Type specific implementation
	err = rs.HandleMediaType(mediaType, req, record)
	if err != nil {
		return err
	}

	// Set Payload
	rs.HandlePayload(req, record)

	// Save Headers to Record
	record.SetHeaders(req.Header)

	err = rs.saveRecord(*record)
	if err != nil {
		return err
	}

	return nil
}

// HandleMediaType takes the supplied mediaType string and performs the necessary actions based on the request.
func (rs *RecordServer) HandleMediaType(mediaType string, req *http.Request, record *records.RqRecord) error {
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
			return StatusError{
				StatusCode: http.StatusInternalServerError,
				Err:        errors.New(errMsg),
			}
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
			return StatusError{
				StatusCode: http.StatusBadRequest,
				Err:        errors.New(errMsg),
			}
		}

		keys, err := rs.HandleFilesInRequest(req)
		if err != nil {
			switch e := err.(type) {
			case HttpError:
				return StatusError{
					StatusCode: e.Status(),
					Err:        err,
				}
			default:
				return StatusError{
					StatusCode: http.StatusInternalServerError,
					Err:        errors.New(http.StatusText(http.StatusInternalServerError)),
				}
			}
		}

		out, _ := json.Marshal(keys)
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
			return StatusError{
				StatusCode: http.StatusInternalServerError,
				Err:        errors.New(errMsg),
			}
		}

	}

	if mediaType == "application/json" {
		body := map[string]json.RawMessage{}
		out, _ := io.ReadAll(req.Body)
		json.Unmarshal(out, &body)
		return nil
	} else {

		url := req.Form.Get("url")
		fmt.Println(req.Form)
		if url == "" {
			errMsg := fmt.Sprintf("no URL provided")
			return StatusError{
				StatusCode: http.StatusBadRequest,
				Err:        errors.New(errMsg),
			}
		}
		record.Url = url
	}

	// Set Content-Type on Record
	record.ContentType = mediaType
	return nil
}

// HandlePayload takes all submitted form key value pairs in the http.Request and saves them to the records.RqRecord
func (rs *RecordServer) HandlePayload(req *http.Request, record *records.RqRecord) {

	payload := map[string][]string{}
	for key, values := range req.Form {
		payload[key] = values
	}

	// remove URL from stored payload, as this isn't sent onwards
	delete(payload, "url")

	out, _ := json.Marshal(payload)
	record.Payload = out
}

// HandleFilesInRequest iterates through all files sent in a request, saves them to disk and returns a slice
// of stings containing the names of all files.
func (rs *RecordServer) HandleFilesInRequest(req *http.Request) (keys []string, err error) {

	rqId := getRqId(req)

	var fileKeys []string
	for key, _ := range req.MultipartForm.File {

		// Store the file on disk
		file, fileHeaders, err := req.FormFile(key)
		if err != nil {
			errMsg := fmt.Sprintf("server error getting file for key: %v, %v", key, err)
			log.Println(errMsg)

			return []string{}, StatusError{
				StatusCode: http.StatusInternalServerError,
				Err:        err,
			}
		}
		srcFileName := fileHeaders.Filename

		fileExtOk, ext := files.CheckExtensionIsAllowed(srcFileName)
		if fileExtOk == false {
			return []string{}, StatusError{
				StatusCode: http.StatusBadRequest,
				Err:        errors.New("File extension not allowed."),
			}
		}

		dstFileName := fmt.Sprintf("%v-%v.%v", rqId, key, ext)

		if err := rs.FileStore.Save(dstFileName, file); err != nil {
			log.Printf("%v: Error saving file %v", rqId, err.Error())
			return []string{}, StatusError{
				StatusCode: http.StatusInternalServerError,
				Err:        err,
			}
		}
		// TODO: Record key names into db
		fileKeys = append(fileKeys, key)

	}
	return fileKeys, nil
}

func (rs *RecordServer) saveRecord(record records.RqRecord) error {
	err := rs.Store.Add(record)
	if err != nil {
		out := fmt.Sprintf("%v: Record save failed: %v", record.Id, err)
		log.Println(out)

		return StatusError{
			StatusCode: http.StatusInternalServerError,
			Err:        err,
		}
	}
	return nil
}

func RqHttpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		rqid := GenerateRequestId()
		ctx := req.Context()
		ctx = context.WithValue(ctx, "rqid", rqid)
		req = req.WithContext(ctx)
		log.Printf("%v: processing request", rqid)
		w.Header().Add("RqId", rqid)
		next.ServeHTTP(w, req)
	})
}

// ReturnHTTPError returns an ErrorResponse back to the client if a request has failed.
func ReturnHTTPErrorResponse(w http.ResponseWriter, errorMessage string, status int) {
	//w.WriteHeader(status)
	output, _ := json.Marshal(ErrorResponse{Error: errorMessage})
	http.Error(w, string(output), status)
}

func GenerateRequestId() string {
	return uuid.New().String()
}

func QueueHttpHandler(w http.ResponseWriter, req *http.Request) {

}

// validateContentType will validate whetheer the value for the Content-Type header value is in the allowed list
func validateContentType(mediaType string) error {
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
	// Check if content-type is not in allowed list
	if helpers.Contains(&config.Config.Server.AllowedContentTypes, mediaType) == false {
		return errors.New("no or unsupported Content-Type supplied")
	}

	// This in theory never happens
	if mediaType == "" {
		return errors.New("no content-type supplied")
	}

	return nil

}

func addServerExcludedHeaders(configHeaders *[]string) {

	serverExcludedHeaders := []string{"Content-Length", "User-Agent", "Content-Type", "Accept"}

	for _, key := range serverExcludedHeaders {
		if helpers.Contains(configHeaders, key) == false {
			config.Config.Server.ExcludedHeaders = append(config.Config.Server.ExcludedHeaders, key)
		}
	}
}

// getRqId returns the RQ ID from the Request's Context
func getRqId(req *http.Request) string {
	rqidCtx := req.Context().Value("rqid")
	rqid := fmt.Sprint(rqidCtx)

	return rqid
}
