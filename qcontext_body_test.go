package vnix

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
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

func TestContextEmptyBody(t *testing.T) {
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
