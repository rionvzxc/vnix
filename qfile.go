package vnix

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
)

type QFile struct {
	fileHeader *multipart.FileHeader
}

// Filename
func (this *QFile) Filename() string {
	if this.fileHeader == nil {
		return ""
	}
	return this.fileHeader.Filename
}

// MimeType
func (this *QFile) MimeType() string {
	if this.fileHeader == nil {
		return ""
	}
	value, _, err := mime.ParseMediaType(this.fileHeader.Header.Get("Content-Type"))
	if err != nil {
		return ""
	}
	return value
}

// Save
func (this *QFile) Save(path string) error {
	if this.fileHeader == nil {
		return http.ErrMissingFile
	}
	file, err := this.fileHeader.Open()

	if err != nil {
		return err
	}
	defer file.Close()
	output, err := os.Create(path)

	if err != nil {
		return err
	}
	defer output.Close()
	_, err = io.Copy(output, file)
	return err
}
