package vnix

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

type tBodyType uint8

const (
	vBodyTypeNone tBodyType = iota
	vBodyTypeRaw
	vBodyTypeText
	vBodyTypeJSON
	vBodyTypeFile
)

var errBodyAlreadyRead = errors.New("request body has already been read")

// Data reads and returns the raw request body once.
func (this *QContext) Data() ([]byte, error) {
	if this.bodyRead {
		return nil, errBodyAlreadyRead
	}
	this.bodyRead = true

	if this.httpRequest.Body == nil {
		return []byte{}, nil
	}
	return io.ReadAll(this.httpRequest.Body)
}

// Text reads and returns the request body as text once.
func (this *QContext) Text() (string, error) {
	if this.bodyRead {
		return "", errBodyAlreadyRead
	}
	this.bodyRead = true

	if this.httpRequest.Body == nil {
		return "", nil
	}
	rawData, err := io.ReadAll(this.httpRequest.Body)
	return string(rawData), err
}

// BindJSON decodes the request body into value once.
func (this *QContext) BindJSON(value any) error {
	if this.bodyRead {
		return errBodyAlreadyRead
	}
	this.bodyRead = true

	if this.httpRequest.Body == nil {
		return io.EOF
	}
	return json.NewDecoder(this.httpRequest.Body).Decode(value)
}

// SetData sets the response body to raw bytes.
func (this *QContext) SetData(data []byte) {
	this.bodyType = vBodyTypeRaw
	this.bodyDataRaw = data
	this.bodyFilePath = ""
	this.resHeaderDefault = make(THeader)
}

// SetText sets the response body to plain text.
func (this *QContext) SetText(data string) {
	this.bodyType = vBodyTypeText
	this.bodyDataText = data
	this.bodyFilePath = ""
	this.resHeaderDefault = THeader{"Content-Type": {"text/plain; charset=utf-8"}}
}

// SetHTML sets the response body with an HTML content type.
func (this *QContext) SetHTML(data string) {
	this.bodyType = vBodyTypeText
	this.bodyDataText = data
	this.bodyFilePath = ""
	this.resHeaderDefault = THeader{"Content-Type": {"text/html; charset=utf-8"}}
}

// SetTextF formats and sets the response body to plain text.
func (this *QContext) SetTextF(format string, args ...any) {
	this.SetText(fmt.Sprintf(format, args...))
}

// SetJSON sets any value as the JSON response body and configures its content type.
func (this *QContext) SetJSON(data any) {
	this.bodyType = vBodyTypeJSON
	this.bodyDataJSON = data
	this.bodyFilePath = ""
	this.resHeaderDefault = THeader{"Content-Type": {"application/json; charset=utf-8"}}
}
