package vnix

import (
	"bytes"
	"errors"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
	if got := ctx.resHeaderDefault.Get("Content-Type"); got != "application/octet-stream" {
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

func TestFileHandlesInvalidOrMissingData(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("malformed"))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	ctx := newContext(request, httptest.NewRecorder(), nil)

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

func TestFailedSetFilePreservesPreviousBodyDefaults(t *testing.T) {
	response := httptest.NewRecorder()
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), response, nil)
	ctx.SetJSON(TMap{"ok": true})
	if err := ctx.SetFile(filepath.Join(t.TempDir(), "missing.txt")); err == nil {
		t.Fatal("missing file should fail")
	}
	if err := ctx.send(); err != nil {
		t.Fatal(err)
	}
	if response.Body.String() != `{"ok":true}` || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("failed setter changed response: (%q, %v)", response.Body.String(), response.Header())
	}
}
