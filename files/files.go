package files

import (
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"rq/config"
	"strings"
)

// FileStore represents a repository capable of accepting a file and saving is.
type FileStore interface {
	Save(filename string, contents io.Reader) error
}

// DiskFileStore is a FileStore for persistant file storage
type DiskFileStore struct {
	store FileStore
}

// InMemoryFileStore is a FileStore for in-memory file storage.
type InMemoryFileStore struct {
	store FileStore
}

// NewDiskFileStore returns a DiskFileStore stuct, for use with persistant file storage.
func NewDiskFileStore() (*DiskFileStore, error) {
	return &DiskFileStore{}, nil
}

// Save creates a file on disk named filename and copies contents into it
func (dfs *DiskFileStore) Save(filename string, contents io.Reader) error {

	path := fmt.Sprintf("%v/%v", config.Config.UploadDirectory, filename)
	out, err := os.Create(path)
	if err != nil {
		log.Println("File save error")
		return err
	}

	io.Copy(out, contents)

	return nil

}

// CheckExtensionIsAllowed checks to see if the filename supplied is an acceptable format,
// and returns a boolean to represent
func CheckExtensionIsAllowed(filename string) (isOk bool, extension string) {
	isOk = false
	extension = ""
	exp := fmt.Sprintf("%v", config.Config.PermittedFileExtensions)
	re := regexp.MustCompile(exp)

	filenameComponents := strings.Split(filename, ".")
	fileNameDotExtension := filenameComponents[len(filenameComponents)-1]

	if re.MatchString(fileNameDotExtension) {
		isOk = true
		extension = fileNameDotExtension
	}

	return isOk, extension

}
