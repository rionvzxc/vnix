package vnix

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type tBodyType uint8

const (
	bodyTypeNone tBodyType = iota
	bodyTypeRaw
	bodyTypeText
	bodyTypeJSON
)
// Body
func (this *QContext) SetData(data []byte) {
	this.bodyType = bodyTypeRaw
	this.bodyDataRaw = data
}
func (this *QContext) SetText(data string) {
	this.bodyType = bodyTypeText
	this.bodyDataText = data
}
func (this *QContext) SetTextF(format string, args ...any) {
	this.SetText(fmt.Sprintf(format, args...))
}
func (this *QContext) SetJSON(data TMap) {
	this.bodyType = bodyTypeJSON
	this.bodyDataJSON = data

	this.resHeader["Content-Type"] = []string{
		"application/json; charset=utf-8",
	}
}
func (this *QContext) SetHeader(header TValues) {
	this.resHeader = header
}
func (this *QContext) SetCookie(cookies []*TCookie) {
	this.resCookies = cookies
}
// Status
func (this *QContext) StatusCode(code int) {
	this.statusCode = code
}
// Send
func (this *QContext) send() error {
	writer := this.httpResponse
	var body []byte

	switch this.bodyType {
	case bodyTypeRaw:
		body = this.bodyDataRaw
	case bodyTypeText:
		body = []byte(this.bodyDataText)
	case bodyTypeJSON:
		var err error
		body, err = json.Marshal(this.bodyDataJSON)

		if err != nil {
			return err
		}
	}
	header := writer.Header()

	for key := range header {
		delete(header, key)
	}
	for key, values := range http.Header(this.resHeader) {
		header[key] = values
	}
	for _, cookie := range this.resCookies {
		http.SetCookie(writer, cookie)
	}
	writer.WriteHeader(this.statusCode)

	if len(body) > 0 {
		_, err := writer.Write(body)
		return err
	}
	return nil
}