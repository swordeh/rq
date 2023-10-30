package main

import (
	"encoding/json"
	"github.com/google/uuid"
	"io"
	"log"
	"net/http"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type RqRequest struct {
	RqId string     `json:"id"`
	File StoredFile `json:"file"`
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

	fileExtOk := CheckExtensionIsAllowed(headers.Filename)
	if !fileExtOk {
		ReturnHTTPErrorResponse(w, "Check file extension is acceptable.", http.StatusBadRequest)
		return
	}

	sf, err := StoreFile(rq.RqId, file)
	if err != nil {
		ReturnHTTPErrorResponse(w, "Failed to save file.", http.StatusInternalServerError)
		return
	}

	rq.File = sf
	output, err := json.Marshal(rq)
	io.WriteString(w, string(output))
}

func main() {

	http.HandleFunc("/api/rq/request", QueueMediaHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))

}
