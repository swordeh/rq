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
	"time"
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

	querystring := req.URL.Query()
	url := querystring.Get("url")

	if url == "" {
		errMsg := fmt.Sprintf("no url supplied")
		ReturnHTTPErrorResponse(w, errMsg, http.StatusBadRequest)
		return
	}

	// Don't create the record until the request is mildly valid
	record := records.RqRecord{
		Id:     rqId,
		Method: req.Method,
		Error:  "",
	}

	record.Url = url
	out, _ := json.Marshal(querystring)

	rqreq := RqRequest{
		Id:     rqId,
		Record: &record,
	}

	record.Payload = out

	switch req.Method {

	case http.MethodGet:

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

	case http.MethodPost, http.MethodPatch, http.MethodPut:
		err := rs.HandleRequest(req, &record)
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

	result, _ := json.Marshal(rqreq)
	io.WriteString(w, string(result))

	return

}

// HandleRequest processes the request made, validates it, saves media and builds up the record to be stored.
func (rs *RecordServer) HandleRequest(req *http.Request, record *records.RqRecord) error {

	// Get RqId
	rqId := getRqId(req)

	// Log request
	log.Printf("%v: Got a %v request", rqId, req.Method)

	// Get the Content-Type full header
	contentTypeHeader := req.Header.Get("Content-Type")

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
	if err = rs.HandleMediaType(mediaType, req, record); err != nil {
		return err
	}

	// Process GET Payloads
	if req.Method == http.MethodGet {
		// Handle request payloads included in req.Form
		rs.HandleQuerystringPayload(req.URL.Query(), record)
	}

	// Process payload for application/json requests
	if mediaType == "application/json" {
		rs.HandleJsonPayload(req.Body, record)
	}

	media_types := []string{"application/x-www-form-urlencoded", "multipart/form-data"}
	if helpers.Contains(&media_types, mediaType) {
		rs.HandleFormPayload(req.Form, record)
	}

	// Save Headers to Record
	headers := map[string][]string{}
	for key, values := range req.Header {
		// If the request header is not in the excluded list, add to the record map
		if helpers.Contains(&config.Config.Server.ExcludedHeaders, key) == false {
			headers[key] = values
		}
	}

	out, _ := json.Marshal(headers)
	record.Headers = out

	record.Status = "PENDING"
	timeNow := time.Now()
	record.CreatedAt = timeNow
	record.UpdatedAt = timeNow

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

	// Set Content-Type on Record
	record.ContentType = mediaType
	return nil
}

// HandleUrl takes the URL from the querystring and adds to the record
func (rs *RecordServer) HandleUrl(url string, record *records.RqRecord) error {
	if url == "" {
		errMsg := fmt.Sprintf("no URL provided")
		return StatusError{
			StatusCode: http.StatusBadRequest,
			Err:        errors.New(errMsg),
		}
	}

	record.Url = url
	return nil
}

// HandlePayload takes all submitted form key value pairs in the http.Request and saves them to the records.RqRecord
func (rs *RecordServer) HandleFormPayload(form map[string][]string, record *records.RqRecord) {
	// remove URL from stored payload, as this isn't sent onwards
	delete(form, "url")

	out, _ := json.Marshal(form)
	record.Payload = out
}

// HandleJsonPayload extracts the JSON payload from the request and sets it to the `Payload` field of the record.
func (rs *RecordServer) HandleJsonPayload(body io.ReadCloser, record *records.RqRecord) {
	payload := map[string]json.RawMessage{}
	out, _ := io.ReadAll(body)
	json.Unmarshal(out, &payload)

	record.Payload = out
}

// HandleQuerystringPayload takes a querystring map and a pointer to a records.RqRecord and processes the querystring payload.
func (rs *RecordServer) HandleQuerystringPayload(qs map[string][]string, record *records.RqRecord) {
	delete(qs, "url")
	out, _ := json.Marshal(qs)
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
		//fmt.Println("Checking file extension: ", srcFileName)

		fileExtOk, ext := files.CheckExtensionIsAllowed(srcFileName, config.Config.PermittedFileExtensions)
		//fmt.Println("File extension is ok: ", fileExtOk, config.Config.PermittedFileExtensions)
		if fileExtOk == false {
			errMsg := fmt.Sprintf("File extension not allowed: %v", srcFileName)
			return []string{key}, StatusError{
				StatusCode: http.StatusBadRequest,
				Err:        errors.New(errMsg),
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
	output, _ := json.Marshal(ErrorResponse{Error: errorMessage})
	http.Error(w, string(output), status)
}

func GenerateRequestId() string {
	return uuid.New().String()
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

	return nil

}

func addServerExcludedHeaders(configHeaders *[]string) {

	// These are hardcoded to always ensure they are present, as these fields can change between the client and RQ
	serverExcludedHeaders := []string{"Content-Length", "User-Agent", "Content-Type", "Accept"}

	for _, key := range serverExcludedHeaders {
		if helpers.Contains(configHeaders, key) == false {
			*configHeaders = append(*configHeaders, key)
		}
	}

}

// getRqId returns the RQ ID from the Request's Context
func getRqId(req *http.Request) string {
	rqidCtx := req.Context().Value("rqid")
	rqid := fmt.Sprint(rqidCtx)

	return rqid
}
