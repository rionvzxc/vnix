package vnix

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

// File returns the first uploaded file for name.
func (this *QContext) File(name string) (*QFile, error) {
	files, err := this.Files(name)

	if err != nil {
		return nil, err
	}
	return files[0], nil
}

// Files returns all uploaded files for name.
func (this *QContext) Files(name string) ([]*QFile, error) {
	if err := this.parseForm(); err != nil {
		return nil, err
	}
	if this.httpRequest.MultipartForm == nil {
		return nil, http.ErrNotMultipart
	}
	fileHeaders := this.httpRequest.MultipartForm.File[name]

	if len(fileHeaders) == 0 {
		return nil, http.ErrMissingFile
	}
	files := make([]*QFile, len(fileHeaders))

	for i, fileHeader := range fileHeaders {
		files[i] = &QFile{fileHeader: fileHeader}
	}
	return files, nil
}

// SetFile sets a file as the response body and detects its content type from its extension.
func (this *QContext) SetFile(path string) error {
	return this.prepareFileResponse(path, false)
}

// SetDownload sets a file as an attachment and detects its content type from its extension.
func (this *QContext) SetDownload(path string) error {
	return this.prepareFileResponse(path, true)
}

func (this *QContext) prepareFileResponse(path string, download bool) error {
	fileInfo, err := os.Stat(path)

	if err != nil {
		return err
	}
	if !fileInfo.Mode().IsRegular() {
		return fmt.Errorf("vnix: %q is not a regular file", path)
	}
	this.bodyType = vBodyTypeFile
	this.bodyFilePath = path
	contentType := mime.TypeByExtension(filepath.Ext(path))

	if contentType == "" {
		contentType = "application/octet-stream"
	}
	this.resHeaderDefault = THeader{"Content-Type": {contentType}}

	if download {
		this.resHeaderDefault.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
			"filename": filepath.Base(path),
		}))
	}
	return nil
}
