package vnix

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestContextReadsHeaders(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Add("X-Token", "abc")
	request.Header.Add("X-Token", "def")
	ctx := newContext(request, httptest.NewRecorder(), nil)
	if ctx.Header("x-token", "") != "abc" || strings.Join(ctx.HeaderValues("X-Token"), ",") != "abc,def" {
		t.Fatal("header accessors returned unexpected values")
	}
	if ctx.Header("X-Missing", "") != "" || ctx.Header("X-Missing", "fallback") != "fallback" || ctx.HeaderValues("X-Missing") != nil {
		t.Fatal("missing header defaults are incorrect")
	}
}

func TestContextHeaderAccessorsReturnCopies(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header["X-Test"] = []string{"first", "second"}
	ctx := newContext(request, httptest.NewRecorder(), nil)
	request.Header.Set("X-Test", "changed")
	values := ctx.HeaderValues("X-Test")
	values[0] = "changed"
	headers := ctx.RequestHeader()
	headers["X-Test"][0] = "changed"
	headers["X-New"] = []string{"value"}
	if !reflect.DeepEqual(ctx.HeaderValues("X-Test"), []string{"first", "second"}) || ctx.reqHeader.Exist("X-New") {
		t.Fatal("header accessors mutated the original request headers")
	}
}

func TestContextResponseHeaderSetters(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetHeader("X-Removed", "remove")
		headers := THeader{"X-Custom": {"value"}}
		ctx.SetResponseHeader(headers)
		headers["X-Custom"][0] = "changed"
		ctx.SetHeader("X-Changed", "old")
		ctx.SetHeader("X-Changed", "value")
		ctx.AddHeader("X-Changed", "second")
		ctx.SetText("ok")
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	if response.Header().Get("X-Custom") != "value" || response.Header().Get("X-Removed") != "" {
		t.Fatalf("replaced response headers = %v", response.Header())
	}
	if !reflect.DeepEqual(response.Header().Values("X-Changed"), []string{"value", "second"}) {
		t.Fatalf("set/add header values = %v", response.Header().Values("X-Changed"))
	}
}

func TestContextBindHeader(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Add("X-Request-ID", "first")
	request.Header.Add("X-Request-ID", "second")
	ctx := newContext(request, httptest.NewRecorder(), nil)
	var headers struct {
		RequestID []string `header:"X-Request-ID"`
	}
	if err := ctx.BindHeader(&headers); err != nil || strings.Join(headers.RequestID, ",") != "first,second" {
		t.Fatalf("BindHeader() = (%#v, %v)", headers, err)
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
