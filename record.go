package main

import (
	"context"
)

type RqRecord struct {
	Id          string `json:"id"`
	Method      string `json:"method"`
	ContentType string `json:"content_type"`
	Url         string `json:"url"`
	Payload     string `json:"payload"`
	Error       string `json:"error"`
}

func (rr *RqRecord) Build(ctx context.Context, url string, method string) error {
	// Does it have a URL?
	//if url == "" {
	//	return errors.New("no url form value supplied")
	//}
	//
	//rr.Url = url
	//
	//contentType := req.Header.Get("Content-Type")
	//
	//if contentType == "application/json" {
	//	rr.ContentType = contentType
	//} else {
	//	rr.ContentType = ""
	//}
	return nil
}

func (rr *RqRecord) Save() {
	db.Create(&rr)
}
