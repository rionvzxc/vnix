package vnix

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

// QContext provides access to an HTTP request and controls its response.
type QContext struct {
	httpRequest  *http.Request
	httpResponse http.ResponseWriter

	reqHeader      THeader
	reqQuery       TValues
	reqCookie      TValues
	reqParam       TValues
	bodyRead       bool
	bodyFormParsed bool
	bodyFormErr    error

	bodyType     tBodyType
	bodyDataRaw  []byte
	bodyDataText string
	bodyDataJSON any
	bodyFilePath string

	resHeader  THeader
	resCookies []*TCookie

	statusCode int
}
func newContext(hReq *http.Request, hRes http.ResponseWriter, reqParam TValues) *QContext {
	return &QContext{
		httpRequest:  hReq,
		httpResponse: hRes,
		reqHeader:    THeader(hReq.Header).Clone(),
		reqParam:     reqParam,
		resHeader:    make(THeader),
		statusCode:   http.StatusOK,
		bodyType:     vBodyTypeNone,
	}
}
// Param returns the first route parameter value for name or defaultValue.
func (this *QContext) Param(name string, defaultValue string) string {
	return this.reqParam.GetDefault(name, defaultValue)
}
// ParamValues returns a copy of all route parameter values for name.
func (this *QContext) ParamValues(name string) []string {
	return this.reqParam.GetAll(name)
}
// RequestParam returns a clone of all route parameters.
func (this *QContext) RequestParam() TValues {
	return this.reqParam.Clone()
}
// Query returns the first query parameter value for name or defaultValue.
func (this *QContext) Query(name string, defaultValue string) string {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	return this.reqQuery.GetDefault(name, defaultValue)
}
// QueryValues returns a copy of all query parameter values for name.
func (this *QContext) QueryValues(name string) []string {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	return this.reqQuery.GetAll(name)
}
// RequestQuery returns a clone of all query parameters.
func (this *QContext) RequestQuery() TValues {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	return this.reqQuery.Clone()
}
// Header returns the first request header value for name or defaultValue.
func (this *QContext) Header(name string, defaultValue string) string {
	return this.reqHeader.GetDefault(name, defaultValue)
}
// HeaderValues returns a copy of all request header values for name.
func (this *QContext) HeaderValues(name string) []string {
	return this.reqHeader.GetAll(name)
}
// RequestHeader returns a clone of all request headers.
func (this *QContext) RequestHeader() THeader {
	return this.reqHeader.Clone()
}
// SetHeader replaces the response values for name.
func (this *QContext) SetHeader(name string, values ...string) {
	if this.resHeader == nil {
		this.resHeader = make(THeader)
	}
	this.resHeader.Set(name, values...)
}
// AddHeader appends response values for name.
func (this *QContext) AddHeader(name string, values ...string) {
	if this.resHeader == nil {
		this.resHeader = make(THeader)
	}
	this.resHeader.Add(name, values...)
}
// Cookie returns the first request cookie value for name or defaultValue.
func (this *QContext) Cookie(name string, defaultValue string) string {
	this.parseCookies()
	return this.reqCookie.GetDefault(name, defaultValue)
}
// CookieValues returns a copy of all request cookie values for name.
func (this *QContext) CookieValues(name string) []string {
	this.parseCookies()
	return this.reqCookie.GetAll(name)
}
// RequestCookie returns a clone of all request cookies.
func (this *QContext) RequestCookie() TValues {
	this.parseCookies()
	return this.reqCookie.Clone()
}
func (this *QContext) parseCookies() {
	if this.reqCookie == nil {
		this.reqCookie = make(TValues)

		for _, cookie := range this.httpRequest.Cookies() {
			this.reqCookie[cookie.Name] = append(this.reqCookie[cookie.Name], cookie.Value)
		}
	}
}
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
// BindParam binds route parameters into a struct using uri tags.
func (this *QContext) BindParam(value any) error {
	return bindValues(value, "uri", this.reqParam)
}
// BindQuery binds query parameters into a struct using query tags.
func (this *QContext) BindQuery(value any) error {
	values, err := url.ParseQuery(this.httpRequest.URL.RawQuery)

	if err != nil {
		return err
	}
	this.reqQuery = TValues(values)
	return bindValues(value, "query", this.reqQuery)
}
// BindHeader binds request headers into a struct using header tags.
func (this *QContext) BindHeader(value any) error {
	return bindValues(value, "header", TValues(this.reqHeader))
}
// BindForm binds form fields into a struct using form tags.
func (this *QContext) BindForm(value any) error {
	if err := this.parseForm(); err != nil {
		return err
	}
	return bindValues(value, "form", TValues(this.httpRequest.PostForm))
}
// BindCookie binds request cookies into a struct using cookie tags.
func (this *QContext) BindCookie(value any) error {
	this.parseCookies()
	return bindValues(value, "cookie", this.reqCookie)
}
// Form returns the first form value for name or defaultValue.
func (this *QContext) Form(name string, defaultValue string) string {
	if err := this.parseForm(); err != nil {
		return defaultValue
	}
	return TValues(this.httpRequest.PostForm).GetDefault(name, defaultValue)
}
// FormValues returns a copy of all values for a form field.
func (this *QContext) FormValues(name string) []string {
	if err := this.parseForm(); err != nil {
		return nil
	}
	return append([]string(nil), this.httpRequest.PostForm[name]...)
}
// RequestForm returns a clone of all parsed form values, or nil if parsing fails.
func (this *QContext) RequestForm() TValues {
	if err := this.parseForm(); err != nil {
		return nil
	}
	return TValues(this.httpRequest.PostForm).Clone()
}
func (this *QContext) parseForm() error {
	if this.bodyFormParsed {
		return this.bodyFormErr
	}
	if this.bodyRead {
		return errBodyAlreadyRead
	}
	this.bodyRead = true
	this.bodyFormParsed = true
	this.bodyFormErr = this.httpRequest.ParseForm()

	if this.bodyFormErr != nil {
		return this.bodyFormErr
	}
	this.bodyFormErr = this.httpRequest.ParseMultipartForm(32 << 20)

	if this.bodyFormErr == http.ErrNotMultipart {
		this.bodyFormErr = nil
	}
	return this.bodyFormErr
}
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
// SetData sets the response body to raw bytes.
func (this *QContext) SetData(data []byte) {
	this.bodyType = vBodyTypeRaw
	this.bodyDataRaw = data
	this.bodyFilePath = ""
}
// SetText sets the response body to plain text.
func (this *QContext) SetText(data string) {
	this.bodyType = vBodyTypeText
	this.bodyDataText = data
	this.bodyFilePath = ""
}
// SetHTML sets the response body with an HTML content type.
func (this *QContext) SetHTML(data string) {
	this.bodyType = vBodyTypeText
	this.bodyDataText = data
	this.bodyFilePath = ""
	this.resHeader.Set("Content-Type", "text/html; charset=utf-8")
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

	this.resHeader.Set("Content-Type", "application/json; charset=utf-8")
}
// SetFile sets a file as the response body and detects its content type from its extension.
func (this *QContext) SetFile(path string) error {
	return this.setFile(path, false)
}
// SetDownload sets a file as an attachment and detects its content type from its extension.
func (this *QContext) SetDownload(path string) error {
	return this.setFile(path, true)
}
func (this *QContext) setFile(path string, download bool) error {
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
	this.resHeader.Set("Content-Type", contentType)

	if download {
		this.resHeader.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
			"filename": filepath.Base(path),
		}))
	} else {
		this.resHeader.Del("Content-Disposition")
	}
	return nil
}
// SetResponseHeader replaces all response headers with a clone of header.
func (this *QContext) SetResponseHeader(header THeader) {
	this.resHeader = header.Clone()

	if this.resHeader == nil {
		this.resHeader = make(THeader)
	}
}
// SetCookie sets the cookies to add to the response.
func (this *QContext) SetCookie(cookies []*TCookie) {
	this.resCookies = cookies
}
// StatusCode sets the HTTP status code for the response.
func (this *QContext) StatusCode(code int) {
	this.statusCode = code
}
func (this *QContext) send() error {
	writer := this.httpResponse
	var body []byte
	var file *os.File

	switch this.bodyType {
	case vBodyTypeRaw:
		body = this.bodyDataRaw
	case vBodyTypeText:
		body = []byte(this.bodyDataText)
	case vBodyTypeJSON:
		var err error
		body, err = json.Marshal(this.bodyDataJSON)

		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		}
	case vBodyTypeFile:
		var err error
		file, err = os.Open(this.bodyFilePath)

		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		}
		defer file.Close()
	}
	header := writer.Header()

	for key, values := range http.Header(this.resHeader) {
		header[http.CanonicalHeaderKey(key)] = append([]string(nil), values...)
	}
	for _, cookie := range this.resCookies {
		if cookie != nil {
			http.SetCookie(writer, cookie)
		}
	}
	writer.WriteHeader(this.statusCode)

	if file != nil {
		_, err := io.Copy(writer, file)
		return err
	}
	if len(body) > 0 {
		_, err := writer.Write(body)
		return err
	}
	return nil
}
