package vnix

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

type tBodyType uint8

const (
	bodyTypeNone tBodyType = iota
	bodyTypeRaw
	bodyTypeText
	bodyTypeJSON
	bodyTypeFile
)

// ErrBodyAlreadyRead is returned when a body reader is used after another body reader consumed it.
var ErrBodyAlreadyRead = errors.New("request body has already been read")

// QContext provides access to an HTTP request and controls its response.
type QContext struct {
	httpRequest  *http.Request
	httpResponse http.ResponseWriter

	reqHeader      http.Header
	reqQuery       TValues
	reqCookie      TValues
	reqParam       TValues
	reqForm        TValues
	bodyRead       bool
	bodyFormParsed bool

	bodyType     tBodyType
	bodyDataRaw  []byte
	bodyDataText string
	bodyDataJSON TMap
	bodyFilePath string

	resHeader  TValues
	resCookies []*TCookie

	statusCode int
}

func newContext(hReq *http.Request, hRes http.ResponseWriter, reqParam TValues) *QContext {
	return &QContext{
		httpRequest:  hReq,
		httpResponse: hRes,
		reqHeader:    hReq.Header,
		reqParam:     reqParam,
		resHeader:    make(TValues),
		statusCode:   http.StatusOK,
		bodyType:     bodyTypeNone,
	}
}

// Param returns the first route parameter value or defaultValue.
func (this *QContext) Param(name string, defaultValue string) string {
	values := this.reqParam[name]

	if len(values) > 0 {
		return values[0]
	}
	return defaultValue
}

// Params returns all route parameter values for name.
func (this *QContext) Params(name string) []string {
	return this.reqParam[name]
}

// Query returns the first query parameter value or defaultValue.
func (this *QContext) Query(name string, defaultValue string) string {
	values := this.Queries(name)

	if len(values) > 0 {
		return values[0]
	}
	return defaultValue
}

// Queries returns all query parameter values for name.
func (this *QContext) Queries(name string) []string {
	if this.reqQuery == nil {
		this.reqQuery = TValues(this.httpRequest.URL.Query())
	}
	return this.reqQuery[name]
}

// Header returns the first request header value or defaultValue.
func (this *QContext) Header(name string, defaultValue string) string {
	values := this.Headers(name)

	if len(values) > 0 {
		return values[0]
	}
	return defaultValue
}

// Headers returns all request header values for name.
func (this *QContext) Headers(name string) []string {
	if values, exists := this.reqHeader[http.CanonicalHeaderKey(name)]; exists {
		return values
	}
	return nil
}

// Cookie returns the first request cookie value or defaultValue.
func (this *QContext) Cookie(name string, defaultValue string) string {
	values := this.Cookies(name)

	if len(values) > 0 {
		return values[0]
	}
	return defaultValue
}

// Cookies returns all request cookie values for name.
func (this *QContext) Cookies(name string) []string {
	if this.reqCookie == nil {
		this.reqCookie = make(TValues)

		for _, cookie := range this.httpRequest.Cookies() {
			this.reqCookie[cookie.Name] = append(this.reqCookie[cookie.Name], cookie.Value)
		}
	}
	return this.reqCookie[name]
}

// Data reads and returns the raw request body once.
func (this *QContext) Data() ([]byte, error) {
	if this.bodyRead {
		return nil, ErrBodyAlreadyRead
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
		return "", ErrBodyAlreadyRead
	}
	this.bodyRead = true

	if this.httpRequest.Body == nil {
		return "", nil
	}
	rawData, err := io.ReadAll(this.httpRequest.Body)
	return string(rawData), err
}

// JSON decodes the request body into value once.
func (this *QContext) JSON(value any) error {
	if this.bodyRead {
		return ErrBodyAlreadyRead
	}
	this.bodyRead = true

	if this.httpRequest.Body == nil {
		return io.EOF
	}
	return json.NewDecoder(this.httpRequest.Body).Decode(value)
}

// Form returns the first form value for name or defaultValue.
func (this *QContext) Form(name string, defaultValue string) string {
	values := this.Forms(name)

	if len(values) > 0 {
		return values[0]
	}
	return defaultValue
}

// Forms returns all values for a form field, or nil if parsing fails.
func (this *QContext) Forms(name string) []string {
	if err := this.parseForm(); err != nil {
		return nil
	}
	return this.httpRequest.PostForm[name]
}
func (this *QContext) parseForm() error {
	if this.bodyRead && !this.bodyFormParsed {
		return ErrBodyAlreadyRead
	}
	this.bodyRead = true
	this.bodyFormParsed = true
	err := this.httpRequest.ParseMultipartForm(32 << 20)

	if err != nil && err != http.ErrNotMultipart {
		return err
	}
	if this.httpRequest.MultipartForm == nil {
		return this.httpRequest.ParseForm()
	}
	return nil
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
	this.bodyType = bodyTypeRaw
	this.bodyDataRaw = data
	this.bodyFilePath = ""
}

// SetText sets the response body to plain text.
func (this *QContext) SetText(data string) {
	this.bodyType = bodyTypeText
	this.bodyDataText = data
	this.bodyFilePath = ""
}

// SetHTML sets the response body with an HTML content type.
func (this *QContext) SetHTML(data string) {
	this.bodyType = bodyTypeText
	this.bodyDataText = data
	this.bodyFilePath = ""
	this.resHeader["Content-Type"] = []string{"text/html; charset=utf-8"}
}

// SetTextF formats and sets the response body to plain text.
func (this *QContext) SetTextF(format string, args ...any) {
	this.SetText(fmt.Sprintf(format, args...))
}

// SetJSON sets the response body to JSON and configures its content type.
func (this *QContext) SetJSON(data TMap) {
	this.bodyType = bodyTypeJSON
	this.bodyDataJSON = data
	this.bodyFilePath = ""

	this.resHeader["Content-Type"] = []string{
		"application/json; charset=utf-8",
	}
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
	this.bodyType = bodyTypeFile
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

// SetHeader replaces the response headers with header.
func (this *QContext) SetHeader(header TValues) {
	this.resHeader = header
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
	case bodyTypeRaw:
		body = this.bodyDataRaw
	case bodyTypeText:
		body = []byte(this.bodyDataText)
	case bodyTypeJSON:
		var err error
		body, err = json.Marshal(this.bodyDataJSON)

		if err != nil {
			http.Error(writer, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return err
		}
	case bodyTypeFile:
		var err error
		file, err = os.Open(this.bodyFilePath)

		if err != nil {
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
