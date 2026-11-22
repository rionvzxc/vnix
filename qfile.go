package vnix

import (
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
)

// QFile provides access to an uploaded multipart file.
type QFile struct {
	fileHeader *multipart.FileHeader
}

// Filename returns the uploaded file's original name.
func (this *QFile) Filename() string {
	if this.fileHeader == nil {
		return ""
	}
	return this.fileHeader.Filename
}

// MimeType returns the uploaded file's media type, or an empty string if unavailable.
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

// Save writes the uploaded file contents to path.
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
	_, copyErr := io.Copy(output, file)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
