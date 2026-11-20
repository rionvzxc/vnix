package vnix

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func performRequest(handler http.Handler, method string, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
func TestUnicodeRoutes(t *testing.T) {
	tests := []struct {
		name      string
		route     string
		path      string
		paramName string
		want      string
	}{
		{"static", "/café", "/café", "", "matched"},
		{"escaped", "/café", "/caf%C3%A9", "", "matched"},
		{"nested", "/日本/😀/", "/日本/😀/", "", "matched"},
		{"parameter", "/café/:名前", "/café/東京", "名前", "東京"},
		{"wildcard", "/日本/*経路", "/日本/画像/😀.png", "経路", "画像/😀.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := New()
			app.Group("/店舗").GET(tt.route, func(ctx *QContext) {
				ctx.SetText(ctx.Param(tt.paramName, "matched"))
			})
			response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/店舗"+tt.path)
			if response.Code != http.StatusOK || response.Body.String() != tt.want {
				t.Fatalf("response = (%d, %q), want (200, %q)", response.Code, response.Body.String(), tt.want)
			}
		})
	}
}

func TestRouteMethodsAndParameters(t *testing.T) {
	app := New()
	app.DELETE("/items/:id", func(ctx *QContext) {
		ctx.SetText(ctx.Param("id", ""))
	})
	handler := &qHttpHandler{rootNode: app.QNode}

	response := performRequest(handler, http.MethodDelete, "/items/42")

	if response.Code != http.StatusOK || response.Body.String() != "42" {
		t.Fatalf("DELETE response = (%d, %q), want (200, %q)", response.Code, response.Body.String(), "42")
	}
	response = performRequest(handler, http.MethodPut, "/items/42")

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
func TestWildcardRoute(t *testing.T) {
	app := New()
	app.GET("/assets/*path", func(ctx *QContext) {
		ctx.SetText(ctx.Param("path", ""))
	})
	app.PATCH("/assets/*path", func(ctx *QContext) {
		ctx.SetText(ctx.Param("path", "updated"))
	})
	handler := &qHttpHandler{rootNode: app.QNode}

	response := performRequest(handler, http.MethodGet, "/assets/css/site.css")

	if response.Code != http.StatusOK || response.Body.String() != "css/site.css" {
		t.Fatalf("wildcard response = (%d, %q), want (200, %q)", response.Code, response.Body.String(), "css/site.css")
	}
	response = performRequest(handler, http.MethodPost, "/assets/css/site.css")
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wildcard wrong-method status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
func TestWildcardRouteUsesDefaultParameterName(t *testing.T) {
	app := New()
	app.GET("/*", func(ctx *QContext) {
		ctx.SetText(ctx.Param("wildcard", ""))
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/assets/logo.svg")

	if response.Code != http.StatusOK || response.Body.String() != "assets/logo.svg" {
		t.Fatalf("default wildcard response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestStaticRouteTakesPrecedenceOverParameter(t *testing.T) {
	app := New()
	app.GET("/users/:id", func(ctx *QContext) {
		ctx.SetText("parameter")
	})
	app.PUT("/users/:id", func(ctx *QContext) {
		ctx.SetText("updated parameter")
	})
	app.GET("/users/new", func(ctx *QContext) {
		ctx.SetText("static")
	})
	handler := &qHttpHandler{rootNode: app.QNode}

	response := performRequest(handler, http.MethodGet, "/users/new")

	if response.Code != http.StatusOK || response.Body.String() != "static" {
		t.Fatalf("static route response = (%d, %q), want (200, %q)", response.Code, response.Body.String(), "static")
	}
}
func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	app := New()
	app.GET("/known", func(ctx *QContext) {
		ctx.SetText("known")
	})
	handler := &qHttpHandler{rootNode: app.QNode}

	response := performRequest(handler, http.MethodGet, "/missing")

	if response.Code != http.StatusNotFound {
		t.Fatalf("missing route status = %d, want %d", response.Code, http.StatusNotFound)
	}
	response = performRequest(handler, http.MethodPost, "/known")

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
func TestFileServerDirectoryServesFiles(t *testing.T) {
	directory := t.TempDir()
	filePath := filepath.Join(directory, "hello.txt")
	if err := os.WriteFile(filePath, []byte("static file"), 0600); err != nil {
		t.Fatal(err)
	}
	app := New()
	staticNode := app.FileServer("/assets", directory)
	if staticNode.nodeType != vNodeTypeFileServer {
		t.Fatalf("static node type = %d, want vNodeTypeFileServer", staticNode.nodeType)
	}
	handler := &qHttpHandler{rootNode: app.QNode}

	response := performRequest(handler, http.MethodGet, "/assets/hello.txt")
	if response.Code != http.StatusOK || response.Body.String() != "static file" {
		t.Fatalf("static response = (%d, %q)", response.Code, response.Body.String())
	}
	response = performRequest(handler, http.MethodHead, "/assets/hello.txt")
	if response.Code != http.StatusOK || response.Body.Len() != 0 {
		t.Fatalf("static HEAD response = (%d, %q)", response.Code, response.Body.String())
	}
	response = performRequest(handler, http.MethodPost, "/assets/hello.txt")
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("static POST status = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
	response = performRequest(handler, http.MethodGet, "/other/hello.txt")
	if response.Code != http.StatusNotFound {
		t.Fatalf("outside static route status = %d, want %d", response.Code, http.StatusNotFound)
	}
	groupedApp := New()
	groupedApp.Group("/api").FileServer("/assets", directory)
	response = performRequest(&qHttpHandler{rootNode: groupedApp.QNode}, http.MethodGet, "/api/assets/hello.txt")
	if response.Code != http.StatusOK || response.Body.String() != "static file" {
		t.Fatalf("grouped static response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestStaticRejectsUsedRouteNode(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*QNode)
	}{
		{name: "handler", setup: func(node *QNode) { node.GET("/assets", func(*QContext) {}) }},
		{name: "static child", setup: func(node *QNode) { node.Group("/assets/files") }},
		{name: "dynamic child", setup: func(node *QNode) { node.Group("/assets/:name") }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := New()
			test.setup(app.QNode)
			defer func() {
				if recover() == nil {
					t.Fatal("FileServer() should reject a route node already in use")
				}
			}()
			app.FileServer("/assets", t.TempDir())
		})
	}
}
func TestFileServerRejectsDynamicFinalSegment(t *testing.T) {
	for _, routePath := range []string{"/assets/:name", "/assets/*path"} {
		t.Run(routePath, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("FileServer(%q) should panic", routePath)
				}
			}()
			New().FileServer(routePath, t.TempDir())
		})
	}
}
func TestFileServerAllowsDynamicSegmentsBeforeStaticNode(t *testing.T) {
	directory := t.TempDir()
	filePath := filepath.Join(directory, "photo.txt")
	if err := os.WriteFile(filePath, []byte("user photo"), 0600); err != nil {
		t.Fatal(err)
	}
	app := New()
	app.FileServer("/users/:id/photo", directory)
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/users/42/photo/photo.txt")
	if response.Code != http.StatusOK || response.Body.String() != "user photo" {
		t.Fatalf("dynamic parent file response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestCannotRegisterChildBelowFileServerOrWildcard(t *testing.T) {
	tests := []struct {
		name string
		node *QNode
	}{
		{name: "file server", node: New().FileServer("/assets", t.TempDir())},
		{name: "wildcard", node: New().Group("/files/*path")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatal("registering a child route should panic")
				}
			}()
			test.node.Group("/child")
		})
	}
}
func TestGroupedRoutes(t *testing.T) {
	app := New()
	api := app.Group("/api")
	api.GET("/users/:id", func(ctx *QContext) {
		ctx.SetText(ctx.Param("id", ""))
	})
	api.POST("/users/:id", func(ctx *QContext) {
		ctx.SetText(ctx.Param("id", ""))
	})

	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/api/users/7")

	if response.Code != http.StatusOK || response.Body.String() != "7" {
		t.Fatalf("group response = (%d, %q), want (200, %q)", response.Code, response.Body.String(), "7")
	}
}
func TestAllHTTPMethodHelpers(t *testing.T) {
	app := New()
	app.GET("/get", func(ctx *QContext) { ctx.SetText("GET") })
	app.POST("/post", func(ctx *QContext) { ctx.SetText("POST") })
	app.PUT("/put", func(ctx *QContext) { ctx.SetText("PUT") })
	app.DELETE("/delete", func(ctx *QContext) { ctx.SetText("DELETE") })
	app.PATCH("/patch", func(ctx *QContext) { ctx.SetText("PATCH") })
	app.Handle(" get ", "/normalized", func(ctx *QContext) { ctx.SetText("normalized") })
	handler := &qHttpHandler{rootNode: app.QNode}

	tests := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/get", body: "GET"},
		{method: http.MethodPost, path: "/post", body: "POST"},
		{method: http.MethodPut, path: "/put", body: "PUT"},
		{method: http.MethodDelete, path: "/delete", body: "DELETE"},
		{method: http.MethodPatch, path: "/patch", body: "PATCH"},
		{method: http.MethodGet, path: "/normalized", body: "normalized"},
	}
	for _, test := range tests {
		response := performRequest(handler, test.method, test.path)

		if response.Code != http.StatusOK || response.Body.String() != test.body {
			t.Errorf("%s %s = (%d, %q), want (200, %q)", test.method, test.path, response.Code, response.Body.String(), test.body)
		}
	}
}
func TestInvalidRouteDefinitionsPanic(t *testing.T) {
	tests := []struct {
		name     string
		register func(*QNode)
	}{
		{name: "empty parameter", register: func(node *QNode) { node.GET("/:", func(*QContext) {}) }},
		{name: "parameter conflict", register: func(node *QNode) {
			node.GET("/users/:id", func(*QContext) {})
			node.GET("/users/:name", func(*QContext) {})
		}},
		{name: "wildcard conflict", register: func(node *QNode) {
			node.GET("/files/*path", func(*QContext) {})
			node.GET("/files/*rest", func(*QContext) {})
		}},
		{name: "wildcard is not final", register: func(node *QNode) { node.GET("/files/*path/more", func(*QContext) {}) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("invalid route registration should panic")
				}
			}()
			test.register(New().QNode)
		})
	}
}
func TestHandleIgnoresEmptyMethod(t *testing.T) {
	node := New().QNode
	node.setMethodHandler(" ", func(*QContext) {})
	if len(node.handlerFuncMap) != 0 {
		t.Fatalf("empty method should not be registered: %#v", node.handlerFuncMap)
	}
}
func TestUnknownDynamicNodeTypeDoesNotMatch(t *testing.T) {
	root := newNode("", vNodeTypeStatic)
	root.dynamicNode = newNode("unknown", tNodeType(99))
	if node, handler, params := root.getMethodHandler(http.MethodGet, "/value"); node != nil || handler != nil || params != nil {
		t.Fatalf("unknown node match = (%#v, %#v, %#v), want no match", node, handler, params)
	}
}
