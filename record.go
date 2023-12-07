package main

import (
	"context"
	"encoding/json"
)

type RqRecord struct {
	Id          string          `json:"id"`
	Method      string          `json:"method"`
	ContentType string          `json:"content_type"`
	Headers     json.RawMessage `json:"headers"`
	Url         string          `json:"url"`
	FileKeys    string          `json:"file_keys"`
	Payload     json.RawMessage `json:"payload"`
	Error       string          `json:"error"`
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

func (rr *RqRecord) SetHeaders(requestHeaders map[string][]string) {
	headers := map[string][]string{}
	for key, values := range requestHeaders {
		// If the request header is not in the excluded list, add to the record map
		if contains(config.Server.ExcludedHeaders, key) == false {
			headers[key] = values
		}
	}

	out, _ := json.Marshal(headers)
	rr.Headers = out
}

func (rr *RqRecord) Save() {
	db.Create(&rr)
}
