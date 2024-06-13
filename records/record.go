package records

import (
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
	Status      string          `json:"status"`
	Error       string          `json:"error"`
}

type RecordStore interface {
	Add(record RqRecord) error
	Get(id string) (*RqRecord, error)
}
