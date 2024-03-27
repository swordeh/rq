package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"rq/config"
	"rq/files"
	"rq/records"
	"testing"
)

func init() {
	config.Config.PermittedFileExtensions = "mp4|jpg"
}

func TestHandleQuerystringPayload(t *testing.T) {
	tests := []struct {
		name      string
		qs        map[string][]string
		record    *records.RqRecord
		expectVal []byte
	}{
		{
			name:      "Valid: Single Entry Querystring",
			qs:        map[string][]string{"foo": []string{"bar"}},
			record:    &records.RqRecord{},
			expectVal: []byte(`{"foo":["bar"]}`),
		},
		{
			name:      "Valid: Multi Entry Querystring",
			qs:        map[string][]string{"foo": []string{"bar"}, "hello": []string{"world"}},
			record:    &records.RqRecord{},
			expectVal: []byte(`{"foo":["bar"],"hello":["world"]}`),
		},
		{
			name:      "Valid: Multi Value Querystring",
			qs:        map[string][]string{"foo": []string{"bar", "baz"}},
			record:    &records.RqRecord{},
			expectVal: []byte(`{"foo":["bar","baz"]}`),
		},
		{
			name:      "Valid: Url Field Present And Ignored",
			qs:        map[string][]string{"url": []string{"https://google.com"}, "foo": []string{"bar"}},
			record:    &records.RqRecord{},
			expectVal: []byte(`{"foo":["bar"]}`),
		},
		{
			name:      "Valid: Empty Querystring",
			qs:        map[string][]string{},
			record:    &records.RqRecord{},
			expectVal: []byte(`{}`),
		},
	}

	rs := RecordServer{}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rs.HandleQuerystringPayload(test.qs, test.record)
			if string(test.record.Payload) != string(test.expectVal) {
				t.Errorf("Case %s failed: expected %v, but got %v", test.name, string(test.expectVal), string(test.record.Payload))
			}
		})
	}
}

type MockMemoryRecordStore struct {
	db map[string]records.RqRecord
}

func (ms *MockMemoryRecordStore) Add(record records.RqRecord) error {
	ms.db[record.Id] = record
	return nil
}

func (ms *MockMemoryRecordStore) Get(id string) (*records.RqRecord, error) {
	if record, ok := ms.db[id]; ok {
		return &record, nil
	}

	return nil, errors.New("No record found.")
}

func TestRecordServer_HandleQuerystringPayload(t *testing.T) {

	MockRecordStore := &MockMemoryRecordStore{}
	MockFileStore, _ := files.NewInMemoryFileStore()

	type fields struct {
		Store     records.RecordStore
		FileStore files.FileStore
	}
	type args struct {
		qs     map[string][]string
		record *records.RqRecord
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		wants  map[string]interface{}
	}{
		{
			name:   "URL is removed from payload",
			fields: fields{FileStore: MockFileStore, Store: MockRecordStore},
			args: args{
				qs: map[string][]string{
					"url": {"http://www.google.com"},
				},
				record: &records.RqRecord{},
			},
			wants: map[string]interface{}{},
		},
		{
			name:   "No payload",
			fields: fields{FileStore: MockFileStore, Store: MockRecordStore},
			args: args{
				qs:     map[string][]string{},
				record: &records.RqRecord{},
			},
			wants: map[string]interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RecordServer{
				Store:     tt.fields.Store,
				FileStore: tt.fields.FileStore,
			}
			rs.HandleQuerystringPayload(tt.args.qs, tt.args.record)
			out, _ := json.Marshal(tt.args.record)
			fmt.Println(string(out))

			var payload map[string]interface{}
			err := json.Unmarshal(tt.args.record.Payload, &payload)
			if err != nil {
				t.Errorf("Error unmarshalling Record payload")
			}

			if _, found := payload["url"]; found {
				t.Errorf("HandleQuerystringPayload() got = %v, want %v", payload, tt.wants)
			}

		})
	}
}

func TestRecordServer_HandleFilesInRequest(t *testing.T) {

	// Test files are stored in memory
	MockFileStore, _ := files.NewInMemoryFileStore()

	// Create a request body
	requestBody := &bytes.Buffer{}

	// Create multipart writer
	multipartWriter := multipart.NewWriter(requestBody)

	// Write the file to the writer

	fileWriter, err := multipartWriter.CreateFormFile("file", "test.jpg")
	if err != nil {
		t.Errorf("Error creating form file")
	}

	// Write the file contents to the writer
	fileContents := []byte("test file contents")
	_, err = fileWriter.Write(fileContents)
	if err != nil {
		t.Errorf("Error writing file contents")
	}

	// Close the writer
	err = multipartWriter.Close()
	if err != nil {
		t.Errorf("Error closing multipart writer")
	}

	// Create a mock request
	mockRequest := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost", Path: "/api/rq/http"},
		Header: http.Header{"Content-Type": []string{multipartWriter.FormDataContentType()}},
		Body:   io.NopCloser(requestBody),
	}

	// Create bad file
	requestBodyBadExtension := &bytes.Buffer{}
	multipartWriterBadExtension := multipart.NewWriter(requestBodyBadExtension)
	fileWriterBadExtension, err := multipartWriterBadExtension.CreateFormFile("file", "test.exe")
	if err != nil {
		t.Errorf("Error creating bad form file")
	}

	badFileContents := []byte("test file contents")
	_, err = fileWriterBadExtension.Write(badFileContents)

	if err != nil {
		t.Errorf("Error writing bad file contents")
	}

	err = multipartWriterBadExtension.Close()
	if err != nil {
		t.Errorf("Error closing bad multipart writer")
	}

	mockRequestBadFileExtension := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost", Path: "/api/rq/http"},
		Header: http.Header{"Content-Type": []string{multipartWriterBadExtension.FormDataContentType()}},
		Body:   io.NopCloser(requestBodyBadExtension),
	}

	// Parse the request so we can access the form data
	err = mockRequest.ParseMultipartForm(32 << 20)
	if err != nil {
		t.Errorf("Error parsing multipart form")

	}

	// Parse bad file request
	err = mockRequestBadFileExtension.ParseMultipartForm(32 << 20)
	if err != nil {
		t.Errorf("Error parsing bad multipart form")

	}

	type fields struct {
		Store     records.RecordStore
		FileStore files.FileStore
	}
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantKeys []string
		wantErr  bool
	}{
		{
			name:     "one file in request with matching keys",
			fields:   fields{FileStore: MockFileStore, Store: &MockMemoryRecordStore{}},
			args:     args{req: mockRequest},
			wantKeys: []string{"file"},
			wantErr:  false,
		},
		{
			name:     "one file in request with bad extension",
			fields:   fields{FileStore: MockFileStore, Store: &MockMemoryRecordStore{}},
			args:     args{req: mockRequestBadFileExtension},
			wantKeys: []string{"file"},
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs := &RecordServer{
				Store:     tt.fields.Store,
				FileStore: tt.fields.FileStore,
			}

			gotKeys, err := rs.HandleFilesInRequest(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleFilesInRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotKeys, tt.wantKeys) {
				t.Errorf("HandleFilesInRequest() gotKeys = %v, want %v", gotKeys, tt.wantKeys)
			}
		})
	}
}
