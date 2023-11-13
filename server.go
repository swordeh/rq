package main

import (
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

// ReturnHTTPError returns an ErrorResponse back to the client if a request has failed.
func ReturnHTTPErrorResponse(w http.ResponseWriter, errorMessage string, status int) {
	w.WriteHeader(status)
	output, _ := json.Marshal(ErrorResponse{Error: errorMessage})
	io.WriteString(w, string(output))
}

func GenerateRequestId() string {
	return uuid.New().String()
}

func QueueMediaHandler(w http.ResponseWriter, req *http.Request) {

	rqid := GenerateRequestId()
	rq := RqRequest{
		RqId: rqid,
	}

	log.Println("Received a request: ", rq)

	file, headers, err := req.FormFile("file")
	if err != nil {
		ReturnHTTPErrorResponse(w, "No form field with key 'file'", http.StatusBadRequest)
		return
	}

	srcFileName := headers.Filename
	dstFileName := rq.RqId
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

	output, err := json.Marshal(rq)
	io.WriteString(w, string(output))
}

func main() {
	// TODO: Move profile selection to CLI arg / env var
	if err := LoadConfigFile("default"); err != nil {
		log.Fatal("Could not load config file", err)
	}
	http.HandleFunc("/api/rq/request", QueueMediaHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))

}
