package records

import (
	"encoding/json"
	"time"
)

type RqRecord struct {
	Id          string          `json:"id" gorm:"primaryKey" `
	Method      string          `json:"method"`
	ContentType string          `json:"content_type"`
	Headers     json.RawMessage `json:"headers"`
	Url         string          `json:"url"`
	FileKeys    string          `json:"file_keys"`
	Payload     json.RawMessage `json:"payload"`
	Status      string          `json:"status"`
	Error       string          `json:"error"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type RecordStore interface {
	Add(record RqRecord) error
	Get(id string) (*RqRecord, error)
}
