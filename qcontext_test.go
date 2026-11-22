package vnix

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
