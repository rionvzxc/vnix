
package vnix

import (
	"encoding/json"
	"io"
	"net/http"
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
func (this *QContext) GetParam(name string) TStrings {
	if values, exists := this.reqParam[name]; exists {
		return TStrings(values)
	}
	return TStrings{}
}
// Query
func (this *QContext) GetQuery(name string) TStrings {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	if values, exists := this.reqQuery[name]; exists {
		return TStrings(values)
	}
	return TStrings{}
}
// Header
func (this *QContext) GetHeader(name string) TStrings {
	if values, exists := this.reqHeader[http.CanonicalHeaderKey(name)]; exists {
		return TStrings(values)
	}
	return TStrings{}
}
// Cookie
func (this *QContext) GetCookie(name string) TStrings {
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
func (this *QContext) GetData() []byte {
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
func (this *QContext) GetText() string {
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
func (this *QContext) GetJSON(value any) bool {
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
func (this *QContext) GetForm(name string) TStrings {
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
func (this *QContext) GetFile(name string) *QFile {
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