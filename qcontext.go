package vnix

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

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

	resHeaderDefault THeader
	resHeader        THeader
	resCookies       []*TCookie

	statusCode   int
	handlers     []THandlerFunc
	handlerIndex int
	aborted      bool
}
func newContext(hReq *http.Request, hRes http.ResponseWriter, reqParam TValues) *QContext {
	return &QContext{
		httpRequest:      hReq,
		httpResponse:     hRes,
		reqHeader:        THeader(hReq.Header).Clone(),
		reqParam:         reqParam,
		resHeaderDefault: make(THeader),
		resHeader:        make(THeader),
		statusCode:       http.StatusOK,
		bodyType:         vBodyTypeNone,
		handlerIndex:     -1,
	}
}
// StatusCode sets the HTTP status code for the response.
func (this *QContext) StatusCode(code int) {
	this.statusCode = code
}
func (this *QContext) send() error {
	writer := this.httpResponse
	var body []byte
	var file *os.File
	var fileSize int64
	var responseErr error
	bodyAllowed := this.statusCode >= 200 && this.statusCode != http.StatusNoContent && this.statusCode != http.StatusResetContent && this.statusCode != http.StatusNotModified
	bodyType := this.bodyType
	if !bodyAllowed {
		bodyType = vBodyTypeNone
	}

	switch bodyType {
	case vBodyTypeRaw:
		body = this.bodyDataRaw
	case vBodyTypeText:
		body = []byte(this.bodyDataText)
	case vBodyTypeJSON:
		body, responseErr = json.Marshal(this.bodyDataJSON)
	case vBodyTypeFile:
		file, responseErr = os.Open(this.bodyFilePath)
		if responseErr == nil {
			defer file.Close()
			var info os.FileInfo
			info, responseErr = file.Stat()
			if responseErr == nil {
				if !info.Mode().IsRegular() {
					responseErr = fmt.Errorf("vnix: %q is not a regular file", this.bodyFilePath)
				} else {
					fileSize = info.Size()
				}
			}
		}
	}
	if responseErr != nil {
		this.statusCode = http.StatusInternalServerError
		this.SetText(http.StatusText(http.StatusInternalServerError) + "\n")
		body = []byte(this.bodyDataText)
		file = nil
		bodyAllowed = true
	}

	header := writer.Header()
	if bodyAllowed {
		contentLength := int64(len(body))
		if file != nil {
			contentLength = fileSize
		}
		header.Set("Content-Length", strconv.FormatInt(contentLength, 10))
		if len(body) > 0 && header.Get("Content-Type") == "" {
			header.Set("Content-Type", http.DetectContentType(body))
		}
	}
	// Explicit response headers override body defaults regardless of setter order.
	for _, responseHeader := range []THeader{this.resHeaderDefault, this.resHeader} {
		for key, values := range responseHeader {
			header[http.CanonicalHeaderKey(key)] = append([]string(nil), values...)
		}
	}
	// A fallback error body replaces the original response and its length.
	if responseErr != nil {
		header.Del("Content-Encoding")
		header.Set("Content-Length", strconv.Itoa(len(body)))
	}
	for _, cookie := range this.resCookies {
		if cookie != nil {
			http.SetCookie(writer, cookie)
		}
	}
	if this.statusCode < 200 || this.statusCode == http.StatusNoContent {
		header.Del("Content-Length")
	} else if this.statusCode == http.StatusResetContent {
		header.Set("Content-Length", "0")
	}
	writer.WriteHeader(this.statusCode)

	if this.httpRequest.Method == http.MethodHead || !bodyAllowed {
		return responseErr
	}
	if file != nil {
		_, err := io.Copy(writer, file)
		return err
	}
	if len(body) > 0 {
		if _, err := writer.Write(body); err != nil {
			return err
		}
	}
	return responseErr
}
