package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type RqRequest struct {
	RqId string `json:"id"`
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

func QueueMediaHandler(w http.ResponseWriter, req *http.Request) {
	rqidCtx := req.Context().Value("rqid")
	rqid := fmt.Sprint(rqidCtx)

	file, headers, err := req.FormFile("file")
	if err != nil {
		ReturnHTTPErrorResponse(w, "No form field with key 'file'", http.StatusBadRequest)
		return
	}

	srcFileName := headers.Filename
	dstFileName := rqid
	err = StoreFile(dstFileName, srcFileName, file)
	if err != nil {
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

	output, err := json.Marshal(RqRequest{RqId: rqid})
	io.WriteString(w, string(output))
}

func main() {
	// TODO: Move profile selection to CLI arg / env var
	if err := LoadConfigFile("default"); err != nil {
		log.Fatal("Could not load config file", err)
	}

	mux := http.NewServeMux()

	MediaRequestHandler := http.HandlerFunc(QueueMediaHandler)
	mux.Handle("/api/rq/request", RqHttpMiddleware(MediaRequestHandler))

	//http.HandleFunc("/api/rq/request", QueueMediaHandler)
	//htto.HandleFunc("/api/rq/http", HttpOnlyHandler)
	log.Fatal(http.ListenAndServe(":8080", mux))

}
