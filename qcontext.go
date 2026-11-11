
package vnix

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type tBodyType uint8

const (
	bodyTypeNone tBodyType = iota
	bodyTypeRaw
	bodyTypeText
	bodyTypeJSON
)

type QContext struct {
	httpRequest  *http.Request
	httpResponse http.ResponseWriter

	reqHeader	http.Header
	reqQuery     TValues
	reqCookie    TValues
	reqParam     TValues
	reqForm	  TValues
	bodyRead  bool

	bodyType     tBodyType
	bodyDataRaw  []byte
	bodyDataText string
	bodyDataJSON TMap

	resHeader  TValues
	resCookies []*TCookie

	statusCode int
}

func newContext(hReq *http.Request, hRes http.ResponseWriter, reqParam TValues) *QContext {
	return &QContext{
		httpRequest: hReq,
		httpResponse: hRes,
		reqHeader: hReq.Header,
		reqParam:      reqParam,
		statusCode:   http.StatusOK,
		bodyType:     bodyTypeNone,
	}
}
// Param
func (this *QContext) Param(name string) TStrings {
	if values, exists := this.reqParam[name]; exists {
		return TStrings(values)
	}
	return TStrings{}
}
// Query
func (this *QContext) Query(name string) TStrings {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	if values, exists := this.reqQuery[name]; exists {
		return TStrings(values)
	}
	return TStrings{}
}
// Header
func (this *QContext) Header(name string) TStrings {
	if values, exists := this.reqHeader[http.CanonicalHeaderKey(name)]; exists {
		return TStrings(values)
	}
	return TStrings{}
}
// Cookie
func (this *QContext) Cookie(name string) TStrings {
	if this.reqCookie == nil {
		this.reqCookie = make(TValues)

		for _, cookie := range this.httpRequest.Cookies() {
			this.reqCookie[cookie.Name] = append(this.reqCookie[cookie.Name], cookie.Value)
		}
	}
	if values, exists := this.reqCookie[name]; exists {
		return TStrings(values)
	}
	return TStrings{}
}
// GetRaw
func (this *QContext) Data() []byte {
	if this.bodyRead {
		return nil
	}
	this.bodyRead = true

	if this.httpRequest.Body == nil {
		return []byte{}
	}
	rawData, err := io.ReadAll(this.httpRequest.Body)

	if err != nil {
		return nil
	}
	return rawData
}
// GetText
func (this *QContext) Text() string {
	if this.bodyRead {
		return ""
	}
	this.bodyRead = true

	if this.httpRequest.Body == nil {
		return ""
	}
	rawData, err := io.ReadAll(this.httpRequest.Body)

	if err != nil {
		return ""
	}
	return string(rawData)
}
// GetJSON
func (this *QContext) JSON(value any) bool {
	if this.bodyRead {
		return false
	}
	this.bodyRead = true

	if this.httpRequest.Body == nil {
		return false
	}
	if err := json.NewDecoder(this.httpRequest.Body).Decode(value); err != nil {
		return false
	}
	return true
}
// BodyForm
func (this *QContext) Form(name string) TStrings {
	err := this.httpRequest.ParseMultipartForm(32 << 20)

	if (err != nil && err != http.ErrNotMultipart) ||
		(this.httpRequest.MultipartForm == nil && this.httpRequest.ParseForm() != nil) {
		return TStrings{}
	}
	this.bodyRead = true

	if values, exist := this.httpRequest.PostForm[name]; exist {
		return TStrings(values)
	}
	return TStrings{}
}
// BodyFile
func (this *QContext) File(name string) *QFile {
	err := this.httpRequest.ParseMultipartForm(32 << 20)

	if err != nil || err != http.ErrNotMultipart || this.httpRequest.MultipartForm == nil {
		return &QFile{}
	}
	this.bodyRead = true
	files := this.httpRequest.MultipartForm.File[name]

	if len(files) == 0 {
		return &QFile{}
	}
	return &QFile{
		fileHeader: files[0],
	}
}
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