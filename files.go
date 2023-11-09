package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
)

var FileExtError = errors.New("invalid file extension")

// TODO: move into config file
const UPLOAD_DIR = "/Users/sam/rq-uploads"
const PERMITTED_FILE_EXTENSIONS = "mp4|jpg" // extensions are seperated by pipes for use in a regex string

// CheckExtensionIsAllowed checks to see if the filename supplied is an acceptable format,
// and returns a boolean to represent
func CheckExtensionIsAllowed(filename string) bool {

	match := false
	exp := fmt.Sprintf(".*\\.(%v)", PERMITTED_FILE_EXTENSIONS)
	re := regexp.MustCompile(exp)

	if re.MatchString(filename) {
		match = true
	}

	return match

}

func StoreFile(filename string, sourceFileName string, file io.Reader) error {

	fileExtOk := CheckExtensionIsAllowed(sourceFileName)
	if !fileExtOk {
		return FileExtError
	}

	path := fmt.Sprintf("%v/%v", UPLOAD_DIR, filename)
	out, err := os.Create(path)

	if err != nil {
		log.Printf("Local file creation failed: %v", err)
		return err
	}

	_, err = io.Copy(out, file)
	if err != nil {
		log.Println("Local file copy failed: ", err)
		return err
	}

	return nil
}
