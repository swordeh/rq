package records

import (
	"encoding/json"
	"rq/config"
	"rq/helpers"
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

type RecordStore interface {
	Add(record RqRecord) error
}

// SetHeaders takes the headers from the request and adds to the Record , providing they are not in the config's
// excluded list.
func (rr *RqRecord) SetHeaders(requestHeaders map[string][]string) {
	headers := map[string][]string{}
	for key, values := range requestHeaders {
		// If the request header is not in the excluded list, add to the record map
		if helpers.Contains(&config.Config.Server.ExcludedHeaders, key) == false {
			headers[key] = values
		}
	}

	out, _ := json.Marshal(headers)
	rr.Headers = out
}
