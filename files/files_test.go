package files

import (
	"fmt"
	"rq/config"
	"rq/records"
	"testing"
)

type InMemoryRecordStore struct {
	db []records.RqRecord
}

func (rs *InMemoryRecordStore) Add(record records.RqRecord) {
	rs.db = append(rs.db, record)
}

func TestCheckExtensionIsAllowed(t *testing.T) {
	config.Config.PermittedFileExtensions = "mp4|jpg"
	var tests = []struct {
		filename   string
		wantBool   bool
		wantString string
	}{
		{"file.mp4", true, "mp4"},
		{"file.jpg.mp4", true, "mp4"},
		{"file.jpg.mp4.exe", false, "exe"},
		{"file.jpg", true, "jpg"},
		{"file.exe", false, "exe"},
		{"file.sh", false, "sh"},
		{"file", false, "file"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%v", tt.filename)
		t.Run(testname, func(t *testing.T) {
			ans, ext := CheckExtensionIsAllowed(tt.filename, config.Config.PermittedFileExtensions)
			if ans != tt.wantBool || ext != tt.wantString {
				t.Errorf("CheckExtensionIsAllowed(\"%v\") = %v, \"%v\"; want %v, %v",
					tt.filename, ans, ext, tt.wantBool, tt.wantString)
			}

		})
	}
}
