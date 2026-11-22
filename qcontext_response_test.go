package vnix

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResponseBodyDefaultsAreReplaced(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "report.txt")
	if err := os.WriteFile(filePath, []byte("report"), 0600); err != nil {
		t.Fatal(err)
	}
	setters := []struct {
		name        string
		set         func(*QContext)
		contentType string
		download    bool
	}{
		{"data", func(ctx *QContext) { ctx.SetData([]byte{0, 1}) }, "", false},
		{"text", func(ctx *QContext) { ctx.SetText("text") }, "text/plain; charset=utf-8", false},
		{"html", func(ctx *QContext) { ctx.SetHTML("<h1>hello</h1>") }, "text/html; charset=utf-8", false},
		{"json", func(ctx *QContext) { ctx.SetJSON(TMap{"ok": true}) }, "application/json; charset=utf-8", false},
		{"file", func(ctx *QContext) {
			if err := ctx.SetFile(filePath); err != nil {
				t.Fatal(err)
			}
		}, "text/plain; charset=utf-8", false},
		{"download", func(ctx *QContext) {
			if err := ctx.SetDownload(filePath); err != nil {
				t.Fatal(err)
			}
		}, "text/plain; charset=utf-8", true},
	}
	for _, previous := range setters {
		for _, current := range setters {
			t.Run(previous.name+"_to_"+current.name, func(t *testing.T) {
				response := httptest.NewRecorder()
				ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), response, nil)
				ctx.SetHeader("X-Custom", "preserved")
				previous.set(ctx)
				current.set(ctx)
				if got := ctx.resHeaderDefault.Get("Content-Type"); got != current.contentType {
					t.Fatalf("default content type = %q, want %q", got, current.contentType)
				}
				if current.name == "data" && len(ctx.resHeaderDefault) != 0 {
					t.Fatalf("SetData defaults must be empty: %v", ctx.resHeaderDefault)
				}
				if ctx.resHeader.Exist("Content-Type") || ctx.resHeader.Exist("Content-Disposition") {
					t.Fatal("body defaults leaked into explicit response headers")
				}
				if err := ctx.send(); err != nil {
					t.Fatal(err)
				}
				if response.Header().Get("X-Custom") != "preserved" {
					t.Fatal("explicit response header was lost")
				}
				if (response.Header().Get("Content-Disposition") != "") != current.download {
					t.Fatalf("stale or missing download header: %v", response.Header())
				}
				wantType := current.contentType
				if current.name == "data" {
					wantType = "application/octet-stream"
				}
				if response.Header().Get("Content-Type") != wantType {
					t.Fatalf("content type = %q, want %q", response.Header().Get("Content-Type"), wantType)
				}
			})
		}
	}
}

func TestExplicitResponseHeadersOverrideDefaults(t *testing.T) {
	for _, setBeforeBody := range []bool{false, true} {
		response := httptest.NewRecorder()
		ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), response, nil)
		setHeaders := func() {
			ctx.SetHeader("Content-Type", "application/problem+json")
			ctx.SetHeader("X-Custom", "first")
			ctx.AddHeader("X-Custom", "second")
		}
		if setBeforeBody {
			setHeaders()
		}
		ctx.SetText("old body")
		ctx.SetJSON(TMap{"error": "example"})
		if !setBeforeBody {
			setHeaders()
		}
		if err := ctx.send(); err != nil {
			t.Fatal(err)
		}
		if response.Header().Get("Content-Type") != "application/problem+json" || !reflect.DeepEqual(response.Header().Values("X-Custom"), []string{"first", "second"}) {
			t.Fatalf("explicit header override failed: %v", response.Header())
		}
	}
}

func TestHeadPreservesExplicitContentLength(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) { ctx.SetText("hello") })
	app.Handle(http.MethodHead, "/", func(ctx *QContext) {
		ctx.SetHeader("Content-Type", "text/plain; charset=utf-8")
		ctx.SetHeader("Content-Length", "5")
	})
	server := httptest.NewServer(&qHttpHandler{rootNode: app.QNode})
	defer server.Close()
	response, err := server.Client().Head(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.ContentLength != 5 || len(body) != 0 {
		t.Fatalf("HEAD length = %d, body = %q, error = %v", response.ContentLength, body, err)
	}
}

func TestResetContentSkipsBodyPreparation(t *testing.T) {
	app := New()
	app.POST("/", func(ctx *QContext) {
		ctx.StatusCode(http.StatusResetContent)
		ctx.SetJSON(make(chan int))
		ctx.SetHeader("Content-Length", "99")
	})
	server := httptest.NewServer(&qHttpHandler{rootNode: app.QNode})
	defer server.Close()
	response, err := server.Client().Post(server.URL, "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != http.StatusResetContent || response.ContentLength != 0 || len(body) != 0 {
		t.Fatalf("205 response = (%d, length %d, body %q, error %v)", response.StatusCode, response.ContentLength, body, err)
	}
}

func TestResponseErrorPreservesExplicitHeaders(t *testing.T) {
	response := httptest.NewRecorder()
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), response, nil)
	ctx.SetHeader("X-Request-ID", "example")
	ctx.SetJSON(make(chan int))
	if err := ctx.send(); err == nil {
		t.Fatal("expected JSON serialization error")
	}
	if response.Code != http.StatusInternalServerError || response.Header().Get("X-Request-ID") != "example" || response.Header().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("error response = (%d, %v)", response.Code, response.Header())
	}
}

func TestErrorResponseReplacesExplicitContentLength(t *testing.T) {
	for _, mode := range []string{"missing file", "invalid JSON"} {
		t.Run(mode, func(t *testing.T) {
			filePath := filepath.Join(t.TempDir(), "file.txt")
			if err := os.WriteFile(filePath, []byte("hello"), 0600); err != nil {
				t.Fatal(err)
			}
			app := New()
			app.GET("/", func(ctx *QContext) {
				ctx.SetHeader("Content-Length", "5")
				ctx.SetHeader("X-Request-ID", "preserved")
				if mode == "invalid JSON" {
					ctx.SetJSON(make(chan int))
					return
				}
				if err := ctx.SetFile(filePath); err != nil {
					t.Error(err)
					return
				}
				if err := os.Remove(filePath); err != nil {
					t.Error(err)
				}
			})
			server := httptest.NewServer(&qHttpHandler{rootNode: app.QNode})
			defer server.Close()
			response, err := server.Client().Get(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusInternalServerError || string(body) != "Internal Server Error\n" || response.ContentLength != int64(len(body)) {
				t.Fatalf("error response = (%d, length %d, body %q)", response.StatusCode, response.ContentLength, body)
			}
			if response.Header.Get("X-Request-ID") != "preserved" {
				t.Fatal("explicit response header was lost")
			}
		})
	}
}

func TestErrorResponseClearsContentEncoding(t *testing.T) {
	var compressed bytes.Buffer
	compressor := gzip.NewWriter(&compressed)
	if _, err := compressor.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := compressor.Close(); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(t.TempDir(), "asset.txt.gz")
	if err := os.WriteFile(filePath, compressed.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	app := New()
	app.GET("/", func(ctx *QContext) {
		if err := ctx.SetFile(filePath); err != nil {
			t.Error(err)
			return
		}
		ctx.SetHeader("Content-Encoding", "gzip")
		if ctx.Query("fail", "") == "true" {
			if err := os.Remove(filePath); err != nil {
				t.Error(err)
			}
		}
	})
	server := httptest.NewServer(&qHttpHandler{rootNode: app.QNode})
	defer server.Close()
	for _, fail := range []bool{false, true} {
		url := server.URL
		if fail {
			url += "?fail=true"
		}
		request, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		// Keep the response encoded so its actual headers can be inspected.
		request.Header.Set("Accept-Encoding", "gzip")
		response, err := server.Client().Do(request)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if fail {
			if response.StatusCode != http.StatusInternalServerError || response.Header.Get("Content-Encoding") != "" || string(body) != "Internal Server Error\n" || response.ContentLength != int64(len(body)) {
				t.Fatalf("fallback response = (%d, %v, %q)", response.StatusCode, response.Header, body)
			}
		} else if response.StatusCode != http.StatusOK || response.Header.Get("Content-Encoding") != "gzip" || !bytes.Equal(body, compressed.Bytes()) {
			t.Fatalf("gzip response = (%d, %v, %q)", response.StatusCode, response.Header, body)
		}
	}
}

func TestReplacingExplicitHeadersPreservesBodyDefaults(t *testing.T) {
	response := httptest.NewRecorder()
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), response, nil)
	ctx.SetJSON(TMap{"ok": true})
	ctx.SetHeader("Content-Type", "application/problem+json")
	ctx.SetResponseHeader(nil)
	if err := ctx.send(); err != nil {
		t.Fatal(err)
	}
	if response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("body default was cleared: %v", response.Header())
	}
}
