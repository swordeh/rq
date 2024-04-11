package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"rq/config"
	"rq/files"
	"rq/records"
	"strings"
	"testing"
)

func init() {
	config.Config.PermittedFileExtensions = "mp4|jpg"
	config.Config.Server.AllowedContentTypes = []string{"application/json"}
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rs := &RecordServer{
				Store:     test.fields.Store,
				FileStore: test.fields.FileStore,
			}
			rs.HandleQuerystringPayload(test.args.qs, test.args.record)
			out, _ := json.Marshal(test.args.record)
			fmt.Println(string(out))

			var payload map[string]interface{}
			err := json.Unmarshal(test.args.record.Payload, &payload)
			if err != nil {
				t.Errorf("Error unmarshalling Record payload")
			}

			if _, found := payload["url"]; found {
				t.Errorf("HandleQuerystringPayload() got = %v, want %v", payload, test.wants)
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rs := &RecordServer{
				Store:     test.fields.Store,
				FileStore: test.fields.FileStore,
			}

			gotKeys, err := rs.HandleFilesInRequest(test.args.req)
			if (err != nil) != test.wantErr {
				t.Errorf("HandleFilesInRequest() error = %v, wantErr %v", err, test.wantErr)
				return
			}
			if !reflect.DeepEqual(gotKeys, test.wantKeys) {
				t.Errorf("HandleFilesInRequest() gotKeys = %v, want %v", gotKeys, test.wantKeys)
			}
		})
	}
}

func NewMockRequestWithFile(filename string, fileContents []byte) (*http.Request, error) {
	// Create a request body
	requestBody := &bytes.Buffer{}

	// Create multipart writer
	multipartWriter := multipart.NewWriter(requestBody)

	// Write the file to the writer
	fileWriter, err := multipartWriter.CreateFormFile("file", filename)
	if err != nil {
		return &http.Request{}, errors.New("Error creating form file")
	}

	// Write the file contents to the writer
	_, err = fileWriter.Write(fileContents)
	if err != nil {
		return &http.Request{}, errors.New("Error writing file contents")
	}

	// Close the writer
	err = multipartWriter.Close()
	if err != nil {
		return &http.Request{}, errors.New("Error closing multipart writer")
	}

	// Create a mock request
	mockRequest := &http.Request{
		Method: http.MethodPost,
		URL:    &url.URL{Scheme: "http", Host: "localhost", Path: "/api/rq/http"},
		Header: http.Header{"Content-Type": []string{multipartWriter.FormDataContentType()}},
		Body:   io.NopCloser(requestBody),
	}

	return mockRequest, nil
}

func NewMockRequestWithUrlEncodedValues(requestUrl string) (*http.Request, error) {
	// Create the form data
	mockFormData := url.Values{}
	mockFormData.Set("mode", "test")

	mockRequest, err := http.NewRequest(http.MethodPost, requestUrl, strings.NewReader(mockFormData.Encode()))
	if err != nil {
		return &http.Request{}, errors.New("creating request error")
	}

	mockRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return mockRequest, nil
}

func NewMockRequestWithBadUrlEncodedValues(requestUrl string) (*http.Request, error) {
	mockRequest, err := http.NewRequest(http.MethodPost, requestUrl, strings.NewReader("%")) // % is erroneous
	if err != nil {
		return &http.Request{}, errors.New("creating request error")
	}

	mockRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	return mockRequest, nil
}

func TestRecordServer_HandleMediaType(t *testing.T) {

	// Test structs
	type fields struct {
		Store     records.RecordStore
		FileStore files.FileStore
	}
	type args struct {
		mediaType string
		req       *http.Request
		record    *records.RqRecord
	}

	// Test files are stored in memory
	mockFileStore, _ := files.NewInMemoryFileStore()
	mockFileContents := []byte("an image")
	mockRequest, err := NewMockRequestWithFile("image.jpg", mockFileContents)
	if err != nil {
		t.Error(err)
	}
	mockRequestWithUrlEncodedValues, err := NewMockRequestWithUrlEncodedValues("https://www.imagination.com")

	if err != nil {
		t.Error(err)
	}

	mockRequestWithBadUrlEncodedValues, err := NewMockRequestWithBadUrlEncodedValues("https://www.imagination.com")
	if err != nil {
		t.Error(err)
	}

	mockRecord := &records.RqRecord{}
	mockRecordUrlEncodedValues := &records.RqRecord{}

	// Actual tests
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "a multipart upload",
			fields: fields{
				Store:     &MockMemoryRecordStore{},
				FileStore: mockFileStore,
			},
			args: args{
				mediaType: "multipart/form-data",
				req:       mockRequest,
				record:    mockRecord,
			},
			wantErr: false,
		},
		{
			name: "a urlencoded form",
			fields: fields{
				Store:     &MockMemoryRecordStore{},
				FileStore: mockFileStore,
			},
			args: args{
				mediaType: "application/x-www-form-urlencoded",
				req:       mockRequestWithUrlEncodedValues,
				record:    mockRecordUrlEncodedValues,
			},
			wantErr: false,
		},
		{
			name: "a urlencoded form with errors",
			fields: fields{
				Store:     &MockMemoryRecordStore{},
				FileStore: mockFileStore,
			},
			args: args{
				mediaType: "application/x-www-form-urlencoded",
				req:       mockRequestWithBadUrlEncodedValues,
				record:    mockRecordUrlEncodedValues,
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rs := &RecordServer{
				Store:     test.fields.Store,
				FileStore: test.fields.FileStore,
			}
			if err := rs.HandleMediaType(test.args.mediaType, test.args.req, test.args.record); (err != nil) != test.wantErr {
				t.Errorf("HandleMediaType() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestRecordServer_HandleUrl(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedError bool
	}{
		{
			name:          "empty url",
			url:           "",
			expectedError: true,
		},
		{
			name:          "valid url",
			url:           "http://example.com",
			expectedError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &MockMemoryRecordStore{}
			fileStore, _ := files.NewInMemoryFileStore()

			server := &RecordServer{
				Store:     store,
				FileStore: fileStore,
			}
			record := &records.RqRecord{}
			err := server.HandleUrl(test.url, record)

			if (err != nil) != test.expectedError {
				t.Errorf("HandleUrl() error = %v, expectedError = %v", err, test.expectedError)
			}

			if !test.expectedError && record.Url != test.url {
				t.Errorf("HandleUrl() url = %v, expectedUrl = %v", record.Url, test.url)
			}
		})
	}
}

func TestRecordServer_HandleFormPayload(t *testing.T) {
	// Initialize the RecordServer we will test
	rs := RecordServer{}

	tests := []struct {
		name     string
		input    map[string][]string
		expected map[string][]string
	}{
		{
			name:     "empty form",
			input:    map[string][]string{},
			expected: map[string][]string{},
		},
		{
			name:     "form with url",
			input:    map[string][]string{"url": {"http://example.com"}},
			expected: map[string][]string{},
		},
		{
			name:     "form with multiple values",
			input:    map[string][]string{"url": {"http://example.com"}, "other": {"value1", "value2"}},
			expected: map[string][]string{"other": {"value1", "value2"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := &records.RqRecord{
				Payload: make([]byte, 0),
			}
			rs.HandleFormPayload(test.input, record)

			var result map[string][]string
			json.Unmarshal(record.Payload, &result)

			assert.Equal(t, test.expected, result)
		})
	}
}

func TestRecordServer_HandleJsonPayload(t *testing.T) {
	tests := []struct {
		name     string
		bodyFunc func() io.ReadCloser
		want     []byte
	}{
		{
			name: "valid JSON",
			bodyFunc: func() io.ReadCloser {
				jsonStr := `{"foo":"bar"}`
				return io.NopCloser(bytes.NewBuffer([]byte(jsonStr)))
			},
			want: []byte(`{"foo":"bar"}`),
		},
		{
			name: "empty body",
			bodyFunc: func() io.ReadCloser {
				jsonStr := ``
				return io.NopCloser(bytes.NewBuffer([]byte(jsonStr)))
			},
			want: []byte(``),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rs := RecordServer{}
			record := &records.RqRecord{}

			rs.HandleJsonPayload(test.bodyFunc(), record)

			got := record.Payload
			if !bytes.Equal(got, test.want) {
				t.Errorf("HandleJsonPayload() = %v, want %v", string(got), string(test.want))
			}
		})
	}
}

func TestRqHttpMiddleware(t *testing.T) {
	tests := []struct {
		name     string
		next     http.Handler
		validate func(t *testing.T, w *httptest.ResponseRecorder)
	}{
		{
			name: "ExistRqId",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {
				if w.Header().Get("RqId") == "" {
					t.Error("RqId expected in header but wasn't present")
				}
			},
		},
		{
			name: "RqIdInContext",
			next: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Context().Value("rqid") == nil {
					t.Error("rqid expected in context but wasn't present")
				}
			}),
			validate: func(t *testing.T, w *httptest.ResponseRecorder) {},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/", nil)
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			handler := RqHttpMiddleware(test.next)
			handler.ServeHTTP(recorder, req)
			test.validate(t, recorder)
		})
	}
}

func TestValidateContentType(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name       string
		mediaType  string
		wantErr    bool
		errorValue string
	}{
		{
			name:       "With valid content type",
			mediaType:  "application/json",
			wantErr:    false,
			errorValue: "",
		},
		{
			name:       "With invalid content type",
			mediaType:  "invalid/type",
			wantErr:    true,
			errorValue: "no or unsupported Content-Type supplied",
		},
		{
			name:       "With empty content type",
			mediaType:  "",
			wantErr:    true,
			errorValue: "no or unsupported Content-Type supplied",
		},
	}

	// Loop over test cases
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			// Execute the function with the test case parameters
			err := validateContentType(test.mediaType)

			// If error was expected
			if test.wantErr {
				// Check that an error was returned and that it matches the expected value
				if err == nil {
					t.Fatalf("Expected an error but got nil")
				} else if err.Error() != test.errorValue {
					t.Fatalf("Expected error value %q but got %q", test.errorValue, err.Error())
				}
			} else { // If no error was expected
				if err != nil {
					t.Fatalf("Expected no error but got %q", err.Error())
				}
			}
		})
	}
}
