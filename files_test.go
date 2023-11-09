package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestCheckExtensionIsAllowed(t *testing.T) {

	var tests = []struct {
		filename string
		want     bool
	}{
		{"file.mp4", true},
		{"file.jpg", true},
		{"file.exe", false},
		{"file.sh", false},
		{"file", false},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%v", tt.filename)
		t.Run(testname, func(t *testing.T) {
			ans := CheckExtensionIsAllowed(tt.filename)
			if ans != tt.want {
				t.Errorf("CheckExtensionIsAllowed(\"%v\") = %v; want %v", tt.filename, ans, tt.want)
			}
		})
	}

}

func TestStoreFile(t *testing.T) {

	data := []byte("test file")
	reader := bytes.NewReader(data)

	type args struct {
		filename       string
		sourceFilename string
		file           io.Reader
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "good video",
			args:    args{filename: "test.mp4", sourceFilename: "file.mp4", file: reader},
			wantErr: false,
		},
		{
			name:    "bad video",
			args:    args{filename: "test.mp4", sourceFilename: "file.exe", file: reader},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := StoreFile(tt.args.filename, tt.args.sourceFilename, tt.args.file)
			if (err != nil) != tt.wantErr {
				t.Errorf("StoreFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Check the function behaved and actually created the file.
			absoluteFilePath := fmt.Sprintf("%v/%v", UPLOAD_DIR, tt.args.filename)
			if _, err = os.Stat(absoluteFilePath); errors.Is(err, os.ErrNotExist) {
				t.Errorf("StoreFile() error = %v", err)
				return
			}
		})
	}
}
