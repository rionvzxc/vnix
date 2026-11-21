package vnix

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRoutePrefixesAndAllowedMethods(t *testing.T) {
	app := New()
	handler := func(ctx *QContext) { ctx.SetText("ok") }
	app.GET("/api/users", handler)
	app.POST("/api/users", handler)
	app.Handle("DELETE", "/api/users", nil)
	app.GET("/items/:id", handler)
	app.GET("/files/*path", handler)
	app.FileServer("/assets", t.TempDir())
	for _, tt := range []struct {
		path   string
		status int
		allow  string
	}{
		{"/", http.StatusNotFound, ""},
		{"/api", http.StatusNotFound, ""},
		{"/missing", http.StatusNotFound, ""},
		{"/api/users", http.StatusMethodNotAllowed, "GET, POST"},
		{"/items/42", http.StatusMethodNotAllowed, "GET"},
		{"/files/a/b", http.StatusMethodNotAllowed, "GET"},
		{"/assets/file.txt", http.StatusMethodNotAllowed, "GET, HEAD"},
	} {
		t.Run(tt.path, func(t *testing.T) {
			response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodPut, tt.path)
			if response.Code != tt.status || response.Header().Get("Allow") != tt.allow {
				t.Fatalf("response = (%d, Allow %q), want (%d, %q)", response.Code, response.Header().Get("Allow"), tt.status, tt.allow)
			}
		})
	}
}

func TestFileServerHeadOverHTTP(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "hello.txt"), []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(directory, "subdir"), 0700); err != nil {
		t.Fatal(err)
	}
	app := New()
	app.FileServer("/assets", directory)
	server := httptest.NewServer(&qHttpHandler{rootNode: app.QNode})
	defer server.Close()
	for _, path := range []string{"/assets/hello.txt", "/assets/subdir", "/assets/missing.txt"} {
		get, err := server.Client().Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(get.Body)
		get.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		head, err := server.Client().Head(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		headBody, err := io.ReadAll(head.Body)
		head.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if head.StatusCode != get.StatusCode || head.ContentLength != int64(len(body)) || len(headBody) != 0 {
			t.Fatalf("%s HEAD = (%d, length %d, body %q), GET = (%d, %d bytes)", path, head.StatusCode, head.ContentLength, headBody, get.StatusCode, len(body))
		}
	}
}
