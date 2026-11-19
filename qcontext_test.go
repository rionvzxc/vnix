package vnix

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type failingRequestBody struct{}

func (failingRequestBody) Read([]byte) (int, error) {
	return 0, errors.New("read failed")
}
func (failingRequestBody) Close() error {
	return nil
}
func TestSetJSONInitializesHeaders(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetJSON(TMap{"message": "ok"})
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	var body map[string]string

	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["message"] != "ok" {
		t.Fatalf("JSON body = %#v", body)
	}
}
func TestSetJSONAcceptsAnyValue(t *testing.T) {
	type responseData struct {
		Message string `json:"message"`
	}
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetJSON(responseData{Message: "any value"})
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	var body responseData
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Message != "any value" {
		t.Fatalf("JSON body = %#v", body)
	}
}
func TestSetHTMLSetsContentType(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetHTML("<h1>VNIX</h1>")
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")

	if response.Code != http.StatusOK || response.Body.String() != "<h1>VNIX</h1>" {
		t.Fatalf("HTML response = (%d, %q)", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
}
func TestSetFileSetsContentTypeAndStreamsBody(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "main.js")
	wantBody := "console.log('vnix');"
	if err := os.WriteFile(filePath, []byte(wantBody), 0600); err != nil {
		t.Fatal(err)
	}
	app := New()
	app.GET("/", func(ctx *QContext) {
		if err := ctx.SetFile(filePath); err != nil {
			t.Errorf("SetFile() error = %v", err)
		}
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	wantContentType := mime.TypeByExtension(filepath.Ext(filePath))
	if wantContentType == "" {
		wantContentType = "application/octet-stream"
	}
	if response.Code != http.StatusOK || response.Body.String() != wantBody {
		t.Fatalf("file response = (%d, %q)", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != wantContentType {
		t.Fatalf("Content-Type = %q, want %q", got, wantContentType)
	}
	if got := response.Header().Get("Content-Disposition"); got != "" {
		t.Fatalf("Content-Disposition = %q, want empty", got)
	}
}
func TestSetDownloadUsesAttachmentDisposition(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "monthly report.txt")
	wantBody := "monthly report"
	if err := os.WriteFile(filePath, []byte(wantBody), 0600); err != nil {
		t.Fatal(err)
	}
	app := New()
	app.GET("/", func(ctx *QContext) {
		if err := ctx.SetDownload(filePath); err != nil {
			t.Errorf("SetDownload() error = %v", err)
		}
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	if response.Code != http.StatusOK || response.Body.String() != wantBody {
		t.Fatalf("download response = (%d, %q)", response.Code, response.Body.String())
	}
	disposition, params, err := mime.ParseMediaType(response.Header().Get("Content-Disposition"))
	if err != nil || disposition != "attachment" || params["filename"] != "monthly report.txt" {
		t.Fatalf("Content-Disposition = (%q, %#v, %v)", disposition, params, err)
	}
}
func TestSetFileReturnsPathErrors(t *testing.T) {
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), nil)
	if err := ctx.SetFile(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("SetFile() should return an error for a missing path")
	}
	if err := ctx.SetDownload(t.TempDir()); err == nil {
		t.Fatal("SetDownload() should return an error for a directory")
	}
}
func TestSetFileUsesBinaryContentTypeForUnknownExtension(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "asset.vnixunknown")
	if err := os.WriteFile(filePath, []byte("asset"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), nil)
	if err := ctx.SetFile(filePath); err != nil {
		t.Fatal(err)
	}
	if got := ctx.resHeader.Get("Content-Type"); got != "application/octet-stream" {
		t.Fatalf("Content-Type = %q", got)
	}
}
func TestSendReturnsErrorWhenFileDisappears(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "asset.txt")
	if err := os.WriteFile(filePath, []byte("asset"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), nil)
	if err := ctx.SetFile(filePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filePath); err != nil {
		t.Fatal(err)
	}
	if err := ctx.send(); err == nil {
		t.Fatal("send() should return an error when the file cannot be opened")
	}
}
func TestFileDisappearsBeforeResponse(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "asset.txt")
	if err := os.WriteFile(filePath, []byte("asset"), 0600); err != nil {
		t.Fatal(err)
	}
	app := New()
	app.GET("/", func(ctx *QContext) {
		if err := ctx.SetFile(filePath); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(filePath); err != nil {
			t.Fatal(err)
		}
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if response.Body.String() != "Internal Server Error\n" {
		t.Fatalf("body = %q", response.Body.String())
	}
}

func TestFormParseErrorPersists(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{"urlencoded", "application/x-www-form-urlencoded", "name=valid&broken=%ZZ"},
		{"multipart", "multipart/form-data; boundary=test", "--test\r\nInvalid-Header\r\n\r\nvalue\r\n--test--\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", tt.contentType)
			ctx := newContext(request, httptest.NewRecorder(), nil)
			var input struct {
				Name string `form:"name"`
			}
			firstErr := ctx.BindForm(&input)
			if firstErr == nil {
				t.Fatal("BindForm() should reject malformed form data")
			}
			if err := ctx.BindForm(&input); err != firstErr {
				t.Fatalf("second error = %v, want %v", err, firstErr)
			}
			if got := ctx.Form("name", "fallback"); got != "fallback" {
				t.Fatalf("Form() = %q, want fallback", got)
			}
			if ctx.FormValues("name") != nil || ctx.RequestForm() != nil {
				t.Fatal("form accessors should reject partially parsed data")
			}
			if _, err := ctx.Files("upload"); err != firstErr {
				t.Fatalf("Files() error = %v, want %v", err, firstErr)
			}
		})
	}
}

func TestInvalidJSONResponseReturnsServerError(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetJSON(TMap{"invalid": make(chan int)})
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
}
func TestFormReadsURLEncodedData(t *testing.T) {
	form := url.Values{"id": {"42"}, "name": {"vnix"}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	app := New()
	app.POST("/", func(ctx *QContext) {
		id := ctx.Form("id", "")
		name := ctx.Form("name", "Unknown")
		ctx.SetText(id + ":" + name)
	})
	(&qHttpHandler{rootNode: app.QNode}).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "42:vnix" {
		t.Fatalf("form response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestFormAndForms(t *testing.T) {
	form := url.Values{"tag": {"first", "second"}, "empty": {""}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := newContext(request, httptest.NewRecorder(), nil)

	if got := ctx.Form("missing", "fallback"); got != "fallback" {
		t.Fatalf("Form() for missing key = %q", got)
	}
	if got := ctx.Form("empty", "fallback"); got != "" {
		t.Fatalf("Form() for present empty value = %q", got)
	}
	if got := ctx.Form("tag", ""); got != "first" {
		t.Fatalf("Form() = %q, want first", got)
	}
	values := ctx.FormValues("tag")

	if len(values) != 2 || values[0] != "first" || values[1] != "second" {
		t.Fatalf("FormValues() = %#v", values)
	}
	values[0] = "changed"
	if got := ctx.Form("tag", ""); got != "first" {
		t.Fatalf("mutating FormValues() result changed stored form value to %q", got)
	}
	allValues := ctx.RequestForm()
	allValues["tag"][0] = "changed"
	if got := ctx.Form("tag", ""); got != "first" {
		t.Fatalf("mutating RequestForm() result changed stored form value to %q", got)
	}
}
func TestFilesReturnsAllMatchingUploads(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for _, upload := range []struct {
		name string
		data string
	}{{name: "first.txt", data: "first"}, {name: "second.txt", data: "second"}} {
		file, err := writer.CreateFormFile("upload", upload.name)

		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(upload.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx := newContext(request, httptest.NewRecorder(), nil)

	file, err := ctx.File("upload")

	if err != nil || file.Filename() != "first.txt" {
		t.Fatalf("File() = (%#v, %v)", file, err)
	}
	files, err := ctx.Files("upload")

	if err != nil || len(files) != 2 || files[0].Filename() != "first.txt" || files[1].Filename() != "second.txt" {
		t.Fatalf("Files() = (%#v, %v)", files, err)
	}
}
func TestFileReadsMultipartUpload(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("upload", "note.txt")

	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("content")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	app := New()
	app.POST("/", func(ctx *QContext) {
		upload, err := ctx.File("upload")

		if err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText(err.Error())
			return
		}
		ctx.SetText(upload.Filename() + ":" + upload.MimeType())
	})
	(&qHttpHandler{rootNode: app.QNode}).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "note.txt:application/octet-stream" {
		t.Fatalf("file response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestMissingFileIsSafe(t *testing.T) {
	file := &QFile{}

	if file.Filename() != "" || file.MimeType() != "" {
		t.Fatalf("empty QFile should return empty metadata")
	}
	if err := file.Save("unused"); err != http.ErrMissingFile {
		t.Fatalf("Save error = %v, want %v", err, http.ErrMissingFile)
	}
	file = &QFile{fileHeader: &multipart.FileHeader{Filename: "missing.txt"}}

	if err := file.Save("unused"); err == nil {
		t.Fatalf("Save() should return an error for an unavailable upload")
	}
	file = &QFile{fileHeader: &multipart.FileHeader{
		Filename: "bad.txt",
		Header:   textproto.MIMEHeader{"Content-Type": {"text/plain; invalid"}},
	}}
	if got := file.MimeType(); got != "" {
		t.Fatalf("MimeType() for invalid media type = %q", got)
	}
}
func TestContextMissingValuesAndEmptyBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Body = nil
	ctx := newContext(request, httptest.NewRecorder(), nil)

	if got := ctx.Param("missing", ""); got != "" {
		t.Fatalf("missing Param() = %q", got)
	}
	if got := ctx.Param("missing", "fallback"); got != "fallback" {
		t.Fatalf("Param() fallback = %q", got)
	}
	if got := ctx.Query("missing", ""); got != "" {
		t.Fatalf("missing Query() = %q", got)
	}
	if got := ctx.Query("missing", "fallback"); got != "fallback" {
		t.Fatalf("Query() fallback = %q", got)
	}
	if got := ctx.Header("X-Missing", ""); got != "" {
		t.Fatalf("missing Header() = %q", got)
	}
	if got := ctx.Header("X-Missing", "header-default"); got != "header-default" {
		t.Fatalf("Header() fallback = %q", got)
	}
	if got := ctx.Cookie("missing", ""); got != "" {
		t.Fatalf("missing Cookie() = %q", got)
	}
	if got := ctx.Cookie("missing", "cookie-default"); got != "cookie-default" {
		t.Fatalf("Cookie() fallback = %q", got)
	}
	if got := ctx.ParamValues("missing"); got != nil {
		t.Fatalf("missing ParamValues() = %#v", got)
	}
	if got := ctx.QueryValues("missing"); got != nil {
		t.Fatalf("missing QueryValues() = %#v", got)
	}
	if got := ctx.HeaderValues("X-Missing"); got != nil {
		t.Fatalf("missing HeaderValues() = %#v", got)
	}
	if got := ctx.CookieValues("missing"); got != nil {
		t.Fatalf("missing CookieValues() = %#v", got)
	}
	if got, err := ctx.Data(); err != nil || len(got) != 0 {
		t.Fatalf("Data() with nil body = %#v", got)
	}
	ctx = newContext(request, httptest.NewRecorder(), nil)

	if got, err := ctx.Text(); err != nil || got != "" {
		t.Fatalf("Text() with nil body = %q", got)
	}
	ctx = newContext(request, httptest.NewRecorder(), nil)

	if err := ctx.BindJSON(&struct{}{}); !errors.Is(err, io.EOF) {
		t.Fatalf("BindJSON() with nil body error = %v, want EOF", err)
	}
}
func TestFormAndFileHandleInvalidOrMissingData(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("malformed"))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	ctx := newContext(request, httptest.NewRecorder(), nil)

	if got := ctx.FormValues("name"); got != nil {
		t.Fatalf("FormValues() for malformed multipart = %#v", got)
	}
	if got := ctx.Form("name", "fallback"); got != "fallback" {
		t.Fatalf("Form() for malformed multipart = %q", got)
	}
	if got, err := ctx.File("upload"); got != nil || err == nil {
		t.Fatalf("File() for malformed multipart = (%#v, %v)", got, err)
	}
	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=value"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx = newContext(request, httptest.NewRecorder(), nil)

	if got, err := ctx.File("upload"); got != nil || !errors.Is(err, http.ErrNotMultipart) {
		t.Fatalf("File() for non-multipart request = (%#v, %v)", got, err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("other", "value"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request = httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx = newContext(request, httptest.NewRecorder(), nil)

	if got, err := ctx.File("upload"); got != nil || !errors.Is(err, http.ErrMissingFile) {
		t.Fatalf("File() for missing multipart field = (%#v, %v)", got, err)
	}
	if got, err := ctx.Files("upload"); got != nil || !errors.Is(err, http.ErrMissingFile) {
		t.Fatalf("Files() for missing multipart field = (%#v, %v)", got, err)
	}
}
func TestSetDataResponse(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetData([]byte("raw response"))
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")

	if response.Code != http.StatusOK || response.Body.String() != "raw response" {
		t.Fatalf("raw response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestResponseWithoutBody(t *testing.T) {
	app := New()
	app.DELETE("/", func(ctx *QContext) {
		ctx.StatusCode(http.StatusNoContent)
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodDelete, "/")

	if response.Code != http.StatusNoContent || response.Body.Len() != 0 {
		t.Fatalf("empty response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestResponsePreservesExistingHeaders(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetText("ok")
	})
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	response.Header().Set("X-Existing", "preserved")
	(&qHttpHandler{rootNode: app.QNode}).ServeHTTP(response, request)

	if got := response.Header().Get("X-Existing"); got != "preserved" {
		t.Fatalf("existing response header = %q", got)
	}
}
func TestContextReadsQueryHeaderAndCookies(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetText(strings.Join([]string{
			ctx.Query("tag", ""),
			strings.Join(ctx.QueryValues("tag"), ","),
			ctx.Query("missing", "default"),
			ctx.Header("X-Token", ""),
			strings.Join(ctx.HeaderValues("X-Token"), ","),
			ctx.Cookie("session", ""),
			strings.Join(ctx.CookieValues("session"), ","),
		}, ":"))
	})
	request := httptest.NewRequest(http.MethodGet, "/?tag=go&tag=vnix", nil)
	request.Header.Add("X-Token", "abc")
	request.Header.Add("X-Token", "def")
	request.AddCookie(&http.Cookie{Name: "session", Value: "xyz"})
	request.AddCookie(&http.Cookie{Name: "session", Value: "123"})
	response := httptest.NewRecorder()
	(&qHttpHandler{rootNode: app.QNode}).ServeHTTP(response, request)

	want := "go:go,vnix:default:abc:abc,def:xyz:xyz,123"

	if response.Code != http.StatusOK || response.Body.String() != want {
		t.Fatalf("context values response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestContextParamValues(t *testing.T) {
	app := New()
	app.GET("/items/:id/:id", func(ctx *QContext) {
		ctx.SetText(strings.Join([]string{
			ctx.Param("id", ""),
			strings.Join(ctx.ParamValues("id"), ","),
			ctx.Param("missing", "default"),
		}, ":"))
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/items/1/2")

	if response.Code != http.StatusOK || response.Body.String() != "1:1,2:default" {
		t.Fatalf("parameter response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestContextMultiValueAccessorsReturnCopies(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?tag=first&tag=second", nil)
	ctx := newContext(request, httptest.NewRecorder(), TValues{"id": {"first", "second"}})
	ctx.reqHeader = THeader{"X-Test": {"first", "second"}}
	ctx.reqQuery = TValues{"tag": {"first", "second"}}
	ctx.reqCookie = TValues{"session": {"first", "second"}}
	accessors := []func() []string{
		func() []string { return ctx.ParamValues("id") },
		func() []string { return ctx.QueryValues("tag") },
		func() []string { return ctx.HeaderValues("X-Test") },
		func() []string { return ctx.CookieValues("session") },
	}
	for _, accessor := range accessors {
		values := accessor()
		values[0] = "changed"
	}
	checks := []struct {
		name string
		got  []string
	}{
		{name: "ParamValues", got: ctx.ParamValues("id")},
		{name: "QueryValues", got: ctx.QueryValues("tag")},
		{name: "HeaderValues", got: ctx.HeaderValues("X-Test")},
		{name: "CookieValues", got: ctx.CookieValues("session")},
	}
	for _, check := range checks {
		if !reflect.DeepEqual(check.got, []string{"first", "second"}) {
			t.Errorf("%s() = %#v, want original values", check.name, check.got)
		}
	}
	allParams := ctx.RequestParam()
	allParams["id"][0] = "changed"
	allParams["new"] = []string{"value"}
	allQueries := ctx.RequestQuery()
	allQueries["tag"][0] = "changed"
	allQueries["new"] = []string{"value"}
	allHeaders := ctx.RequestHeader()
	allHeaders["X-Test"][0] = "changed"
	allHeaders["X-New"] = []string{"value"}
	allCookies := ctx.RequestCookie()
	allCookies["session"][0] = "changed"
	allCookies["new"] = []string{"value"}
	if !reflect.DeepEqual(ctx.ParamValues("id"), []string{"first", "second"}) || ctx.reqParam.Exist("new") {
		t.Error("RequestParam() clone mutated the original route parameters")
	}
	if !reflect.DeepEqual(ctx.QueryValues("tag"), []string{"first", "second"}) || ctx.reqQuery.Exist("new") {
		t.Error("RequestQuery() clone mutated the original query parameters")
	}
	if !reflect.DeepEqual(ctx.HeaderValues("X-Test"), []string{"first", "second"}) || ctx.reqHeader.Exist("X-New") {
		t.Error("RequestHeader() clone mutated the original request headers")
	}
	if !reflect.DeepEqual(ctx.CookieValues("session"), []string{"first", "second"}) || ctx.reqCookie.Exist("new") {
		t.Error("RequestCookie() clone mutated the original cookies")
	}
}
func TestContextReadsBodyOnlyOnce(t *testing.T) {
	t.Run("raw data", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
		ctx := newContext(request, httptest.NewRecorder(), nil)
		got, err := ctx.Data()

		if err != nil || string(got) != "body" {
			t.Fatalf("Data() = %q", got)
		}
		if _, err := ctx.Data(); !errors.Is(err, errBodyAlreadyRead) {
			t.Fatalf("second Data() error = %v, want ErrBodyAlreadyRead", err)
		}
	})

	t.Run("text", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
		ctx := newContext(request, httptest.NewRecorder(), nil)
		got, err := ctx.Text()

		if err != nil || got != "body" {
			t.Fatalf("Text() = %q", got)
		}
		if _, err := ctx.Text(); !errors.Is(err, errBodyAlreadyRead) {
			t.Fatalf("second Text() error = %v, want ErrBodyAlreadyRead", err)
		}
	})

	t.Run("json", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"vnix"}`))
		ctx := newContext(request, httptest.NewRecorder(), nil)
		var value struct {
			Name string `json:"name"`
		}
		if err := ctx.BindJSON(&value); err != nil || value.Name != "vnix" {
			t.Fatalf("BindJSON() = (%v, %#v), error %v", value.Name == "vnix", value, err)
		}
		if err := ctx.BindJSON(&value); !errors.Is(err, errBodyAlreadyRead) {
			t.Fatalf("second BindJSON() error = %v, want ErrBodyAlreadyRead", err)
		}
	})
}
func TestContextReadErrors(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Body = failingRequestBody{}
	ctx := newContext(request, httptest.NewRecorder(), nil)

	if _, err := ctx.Data(); err == nil {
		t.Fatalf("Data() should return a read error")
	}
	request = httptest.NewRequest(http.MethodPost, "/", nil)
	request.Body = failingRequestBody{}
	ctx = newContext(request, httptest.NewRecorder(), nil)

	if _, err := ctx.Text(); err == nil {
		t.Fatalf("Text() should return a read error")
	}
	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid JSON"))
	ctx = newContext(request, httptest.NewRecorder(), nil)

	if err := ctx.BindJSON(&struct{}{}); err == nil {
		t.Fatalf("BindJSON() should return an error for malformed JSON")
	}
}
func TestFormAfterRawBodyReadReturnsDefault(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=value"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := newContext(request, httptest.NewRecorder(), nil)

	if _, err := ctx.Data(); err != nil {
		t.Fatal(err)
	}
	if got := ctx.Form("name", "fallback"); got != "fallback" {
		t.Fatalf("Form() after Data() = %q, want fallback", got)
	}
}
func TestResponseSetters(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetHeader("X-Removed", "remove")
		header := THeader{"X-Custom": {"value"}}
		ctx.SetResponseHeader(header)
		header["X-Custom"][0] = "changed"
		ctx.SetHeader("X-Changed", "old")
		ctx.SetHeader("X-Changed", "value")
		ctx.AddHeader("X-Changed", "second")
		ctx.SetCookie([]*TCookie{{Name: "session", Value: "active"}, nil})
		ctx.StatusCode(http.StatusCreated)
		ctx.SetTextF("item-%d", 7)
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")

	if response.Code != http.StatusCreated || response.Body.String() != "item-7" {
		t.Fatalf("response = (%d, %q)", response.Code, response.Body.String())
	}
	if got := response.Header().Get("X-Custom"); got != "value" {
		t.Fatalf("custom header = %q", got)
	}
	if got := response.Header().Values("X-Changed"); !reflect.DeepEqual(got, []string{"value", "second"}) {
		t.Fatalf("set/add response header values = %#v", got)
	}
	if got := response.Header().Get("X-Removed"); got != "" {
		t.Fatalf("SetResponseHeader() retained old header value %q", got)
	}
	if got := response.Result().Cookies(); len(got) != 1 || got[0].Name != "session" || got[0].Value != "active" {
		t.Fatalf("response cookies = %#v", got)
	}
}
func TestQFileSave(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("upload", "note.txt")

	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("saved content")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	tempDir, err := os.MkdirTemp(".", ".vnix-test-")

	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(tempDir)
	})
	destination := filepath.Join(tempDir, "saved.txt")
	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	ctx := newContext(request, httptest.NewRecorder(), nil)

	uploadFile, err := ctx.File("upload")

	if err != nil {
		t.Fatalf("File() error = %v", err)
	}
	if err := uploadFile.Save(destination); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	saved, err := os.ReadFile(destination)

	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != "saved content" {
		t.Fatalf("saved file = %q", saved)
	}
	if err := uploadFile.Save(filepath.Join(tempDir, "missing", "file.txt")); err == nil {
		t.Fatalf("Save() should return an error for a missing destination directory")
	}
}
