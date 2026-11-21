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
func TestSetTextUsesPlainTextContentType(t *testing.T) {
	for _, body := range []string{"<script>alert(1)</script>", `{"ok":true}`, ""} {
		app := New()
		app.GET("/", func(ctx *QContext) { ctx.SetText(body) })
		response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
		if response.Header().Get("Content-Type") != "text/plain; charset=utf-8" || response.Body.String() != body {
			t.Fatalf("response = (%q, %q), want plain text body %q", response.Header().Get("Content-Type"), response.Body.String(), body)
		}
	}
}

func TestTextReplacesPreviousContentType(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(*QContext)
	}{
		{"json", func(ctx *QContext) { ctx.SetJSON(TMap{"ok": true}) }},
		{"html", func(ctx *QContext) { ctx.SetHTML("<h1>hello</h1>") }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, formatted := range []bool{false, true} {
				app := New()
				app.GET("/", func(ctx *QContext) {
					tt.set(ctx)
					if formatted {
						ctx.SetTextF("error-%d", 1)
					} else {
						ctx.SetText("error-1")
					}
				})
				response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
				if response.Header().Get("Content-Type") != "text/plain; charset=utf-8" || response.Body.String() != "error-1" {
					t.Fatalf("response = (%q, %q), want plain text error-1", response.Header().Get("Content-Type"), response.Body.String())
				}
			}
		})
	}
}

func TestContentTypeCanBeChangedAfterSetText(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(*QContext)
		want string
	}{
		{"json", func(ctx *QContext) { ctx.SetJSON(TMap{"ok": true}) }, "application/json; charset=utf-8"},
		{"html", func(ctx *QContext) { ctx.SetHTML("<h1>hello</h1>") }, "text/html; charset=utf-8"},
		{"custom", func(ctx *QContext) { ctx.SetHeader("Content-Type", "text/csv") }, "text/csv"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := New()
			app.GET("/", func(ctx *QContext) {
				ctx.SetText("initial")
				tt.set(ctx)
			})
			response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
			if response.Header().Get("Content-Type") != tt.want {
				t.Fatalf("Content-Type = %q, want %q", response.Header().Get("Content-Type"), tt.want)
			}
		})
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

func TestHeadResponseContentLength(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(*QContext)
		want string
	}{
		{"text", func(ctx *QContext) { ctx.SetText("café") }, "5"},
		{"json", func(ctx *QContext) { ctx.SetJSON(TMap{"ok": true}) }, "11"},
		{"raw", func(ctx *QContext) { ctx.SetData([]byte{0, 1, 2}) }, "3"},
		{"empty", func(*QContext) {}, "0"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				response := httptest.NewRecorder()
				ctx := newContext(httptest.NewRequest(method, "/", nil), response, nil)
				tt.set(ctx)
				if err := ctx.send(); err != nil {
					t.Fatal(err)
				}
				if got := response.Header().Get("Content-Length"); got != tt.want {
					t.Fatalf("%s length = %q, want %q", method, got, tt.want)
				}
				if method == http.MethodHead && response.Body.Len() != 0 {
					t.Fatal("HEAD response must have no body")
				}
			}
		})
	}
}

func TestBodylessResponseStatuses(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusNotModified} {
		response := httptest.NewRecorder()
		ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), response, nil)
		ctx.StatusCode(status)
		ctx.SetText("must not be sent")
		if status == http.StatusNoContent {
			ctx.SetHeader("Content-Length", "16")
		}
		if err := ctx.send(); err != nil {
			t.Fatal(err)
		}
		if response.Body.Len() != 0 || response.Header().Get("Content-Length") != "" {
			t.Fatalf("status %d body = %q, length = %q", status, response.Body.String(), response.Header().Get("Content-Length"))
		}
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
		ctx.StatusCode(http.StatusCreated)
		ctx.SetTextF("item-%d", 7)
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	if response.Code != http.StatusCreated || response.Body.String() != "item-7" {
		t.Fatalf("response = (%d, %q)", response.Code, response.Body.String())
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
