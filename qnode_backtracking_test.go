package vnix

import (
	"net/http"
	"reflect"
	"testing"
)

func TestBacktrackingFromStaticToParameter(t *testing.T) {
	app := New()
	app.GET("/users/new", func(ctx *QContext) { ctx.SetText("static") })
	app.GET("/users/:id/posts", func(ctx *QContext) { ctx.SetText(ctx.Param("id", "")) })
	handler := &qHttpHandler{rootNode: app.QNode}
	for _, tt := range []struct {
		path string
		body string
	}{
		{"/users/new", "static"},
		{"/users/new/posts", "new"},
		{"/users/42/posts", "42"},
	} {
		response := performRequest(handler, http.MethodGet, tt.path)
		if response.Code != http.StatusOK || response.Body.String() != tt.body {
			t.Fatalf("%s = (%d, %q), want (200, %q)", tt.path, response.Code, response.Body.String(), tt.body)
		}
	}
	response := performRequest(handler, http.MethodGet, "/users/new/missing")
	if response.Code != http.StatusNotFound {
		t.Fatalf("unmatched path status = %d, want 404", response.Code)
	}
}

func TestBacktrackingFromStaticToWildcard(t *testing.T) {
	app := New()
	app.GET("/docs/help", func(ctx *QContext) { ctx.SetText("static") })
	app.GET("/docs/*path", func(ctx *QContext) { ctx.SetText(ctx.Param("path", "")) })
	handler := &qHttpHandler{rootNode: app.QNode}
	for _, tt := range []struct {
		path string
		body string
	}{
		{"/docs/help", "static"},
		{"/docs/help/more", "help/more"},
		{"/docs/other/more", "other/more"},
	} {
		response := performRequest(handler, http.MethodGet, tt.path)
		if response.Code != http.StatusOK || response.Body.String() != tt.body {
			t.Fatalf("%s = (%d, %q), want (200, %q)", tt.path, response.Code, response.Body.String(), tt.body)
		}
	}
}

func TestBacktrackingRestoresParametersAndSelectsMiddleware(t *testing.T) {
	app := New()
	app.GET("/a/static/:id/miss", func(*QContext) { t.Fatal("dead-end handler ran") }).
		UseMiddleware(func(*QContext) { t.Fatal("dead-end middleware ran") })
	app.GET("/a/:id/fixed/:id", func(ctx *QContext) {
		if got := ctx.ParamValues("id"); !reflect.DeepEqual(got, []string{"static", "value"}) {
			t.Fatalf("parameters leaked from dead branch: %v", got)
		}
		ctx.SetText("matched")
	}).UseMiddleware(func(ctx *QContext) { ctx.SetHeader("X-Matched", "yes") })
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/a/static/fixed/value")
	if response.Code != http.StatusOK || response.Body.String() != "matched" || response.Header().Get("X-Matched") != "yes" {
		t.Fatalf("matched response = (%d, %q, %v)", response.Code, response.Body.String(), response.Header())
	}
}

func TestBacktrackingWhenStaticNodeHasNoEndpoint(t *testing.T) {
	app := New()
	app.GET("/items/new/details", func(ctx *QContext) { ctx.SetText("details") })
	app.GET("/items/:id", func(ctx *QContext) { ctx.SetText(ctx.Param("id", "")) })
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/items/new")
	if response.Code != http.StatusOK || response.Body.String() != "new" {
		t.Fatalf("response = (%d, %q), want (200, new)", response.Code, response.Body.String())
	}
}

func TestBacktrackingPreservesMethodNotAllowed(t *testing.T) {
	app := New()
	app.GET("/users/new", func(ctx *QContext) { ctx.SetText("static") })
	app.POST("/users/:id", func(ctx *QContext) { ctx.SetText(ctx.Param("id", "")) })
	app.GET("/users/:id/posts", func(ctx *QContext) { ctx.SetText("posts") })
	handler := &qHttpHandler{rootNode: app.QNode}
	response := performRequest(handler, http.MethodPost, "/users/new")
	if response.Code != http.StatusOK || response.Body.String() != "new" {
		t.Fatalf("method fallback = (%d, %q)", response.Code, response.Body.String())
	}
	response = performRequest(handler, http.MethodPut, "/users/new/posts")
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != "GET" {
		t.Fatalf("method mismatch = (%d, %v)", response.Code, response.Header())
	}
}
