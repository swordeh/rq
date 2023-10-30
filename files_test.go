package main

import (
	"fmt"
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
