package vnix

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
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
	form := url.Values{"name": {"vnix"}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	app := New()
	app.POST("/", func(ctx *QContext) {
		ctx.SetText(ctx.Form("name").String("missing"))
	})
	(&qHttpHandler{rootNode: app.QNode}).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "vnix" {
		t.Fatalf("form response = (%d, %q)", response.Code, response.Body.String())
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
		upload := ctx.File("upload")
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

	if got := ctx.Param("missing").String("fallback"); got != "fallback" {
		t.Fatalf("missing Param() = %q", got)
	}
	if got := ctx.Query("missing").String("fallback"); got != "fallback" {
		t.Fatalf("missing Query() = %q", got)
	}
	if got := ctx.Header("X-Missing").String("fallback"); got != "fallback" {
		t.Fatalf("missing Header() = %q", got)
	}
	if got := ctx.Cookie("missing").String("fallback"); got != "fallback" {
		t.Fatalf("missing Cookie() = %q", got)
	}
	if got := ctx.Data(); len(got) != 0 {
		t.Fatalf("Data() with nil body = %#v", got)
	}
	ctx = newContext(request, httptest.NewRecorder(), nil)
	if got := ctx.Text(); got != "" {
		t.Fatalf("Text() with nil body = %q", got)
	}
	ctx = newContext(request, httptest.NewRecorder(), nil)
	if ctx.JSON(&struct{}{}) {
		t.Fatalf("JSON() with nil body should fail")
	}
}

func TestFormAndFileHandleInvalidOrMissingData(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("malformed"))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	ctx := newContext(request, httptest.NewRecorder(), nil)

	if got := ctx.Form("name"); len(got) != 0 {
		t.Fatalf("Form() for malformed multipart = %#v", got)
	}
	if got := ctx.File("upload"); got.Filename() != "" {
		t.Fatalf("File() for malformed multipart = %q", got.Filename())
	}

	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=value"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx = newContext(request, httptest.NewRecorder(), nil)
	if got := ctx.File("upload"); got.Filename() != "" {
		t.Fatalf("File() for non-multipart request = %q", got.Filename())
	}

	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=%ZZ"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx = newContext(request, httptest.NewRecorder(), nil)
	if got := ctx.Form("name"); len(got) != 0 {
		t.Fatalf("Form() for invalid URL encoding = %#v", got)
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
	if got := ctx.File("upload"); got.Filename() != "" {
		t.Fatalf("File() for missing multipart field = %q", got.Filename())
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
			ctx.Query("tag").String("missing"),
			ctx.Header("X-Token").String("missing"),
			ctx.Cookie("session").String("missing"),
		}, ":"))
	})
	request := httptest.NewRequest(http.MethodGet, "/?tag=go", nil)
	request.Header.Set("X-Token", "abc")
	request.AddCookie(&http.Cookie{Name: "session", Value: "xyz"})
	response := httptest.NewRecorder()
	(&qHttpHandler{rootNode: app.QNode}).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "go:abc:xyz" {
		t.Fatalf("context values response = (%d, %q)", response.Code, response.Body.String())
	}
}

func TestContextReadsBodyOnlyOnce(t *testing.T) {
	t.Run("raw data", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
		ctx := newContext(request, httptest.NewRecorder(), nil)
		if got := string(ctx.Data()); got != "body" {
			t.Fatalf("Data() = %q", got)
		}
		if got := ctx.Data(); got != nil {
			t.Fatalf("second Data() = %q, want nil", got)
		}
	})

	t.Run("text", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
		ctx := newContext(request, httptest.NewRecorder(), nil)
		if got := ctx.Text(); got != "body" {
			t.Fatalf("Text() = %q", got)
		}
		if got := ctx.Text(); got != "" {
			t.Fatalf("second Text() = %q, want empty", got)
		}
	})

	t.Run("json", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"vnix"}`))
		ctx := newContext(request, httptest.NewRecorder(), nil)
		var value struct {
			Name string `json:"name"`
		}
		if !ctx.JSON(&value) || value.Name != "vnix" {
			t.Fatalf("JSON() = (%v, %#v)", value.Name == "vnix", value)
		}
		if ctx.JSON(&value) {
			t.Fatalf("second JSON() should fail after body was read")
		}
	})
}

func TestContextReadErrors(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Body = failingRequestBody{}
	ctx := newContext(request, httptest.NewRecorder(), nil)
	if got := ctx.Data(); got != nil {
		t.Fatalf("Data() after read error = %#v", got)
	}

	request = httptest.NewRequest(http.MethodPost, "/", nil)
	request.Body = failingRequestBody{}
	ctx = newContext(request, httptest.NewRecorder(), nil)
	if got := ctx.Text(); got != "" {
		t.Fatalf("Text() after read error = %q", got)
	}

	request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader("invalid JSON"))
	ctx = newContext(request, httptest.NewRecorder(), nil)
	if ctx.JSON(&struct{}{}) {
		t.Fatalf("JSON() should fail for malformed JSON")
	}
}

func TestResponseSetters(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetHeader(TValues{"X-Custom": {"value"}})
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

	if err := ctx.File("upload").Save(destination); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	saved, err := os.ReadFile(destination)

	if err != nil {
		t.Fatal(err)
	}
	if string(saved) != "saved content" {
		t.Fatalf("saved file = %q", saved)
	}
	if err := ctx.File("upload").Save(filepath.Join(tempDir, "missing", "file.txt")); err == nil {
		t.Fatalf("Save() should return an error for a missing destination directory")
	}
}
