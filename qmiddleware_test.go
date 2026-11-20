package vnix

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMiddlewareConfiguration(t *testing.T) {
	for _, tt := range []struct {
		name    string
		modeSet bool
		inherit bool
		local   []string
		child   []string
	}{
		{"AddLocal", false, false, []string{"root", "node", "handler"}, []string{"root", "handler"}},
		{"AddInherited", false, true, []string{"root", "node", "handler"}, []string{"root", "node", "handler"}},
		{"SetLocal", true, false, []string{"node", "handler"}, []string{"root", "handler"}},
		{"SetInherited", true, true, []string{"node", "handler"}, []string{"node", "handler"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			app := New()
			var calls []string
			record := func(name string) THandlerFunc {
				return func(*QContext) { calls = append(calls, name) }
			}
			node := app.Group("/node")
			node.GET("", record("handler"))
			node.GET("/child", record("handler"))
			// Configure after route registration to verify existing descendants.
			app.UseMiddleware(record("root")).Inherit(true)
			middleware := node.UseMiddleware(record("node"))
			if tt.modeSet {
				middleware.ModeSet()
			} else {
				middleware.ModeAdd()
			}
			middleware.Inherit(tt.inherit)
			handler := &qHttpHandler{rootNode: app.QNode}
			for _, request := range []struct {
				path string
				want []string
			}{{"/node", tt.local}, {"/node/child", tt.child}} {
				calls = nil
				performRequest(handler, http.MethodGet, request.path)
				if !reflect.DeepEqual(calls, request.want) {
					t.Fatalf("%s calls = %v, want %v", request.path, calls, request.want)
				}
			}
		})
	}
}

func TestMiddlewareDefaults(t *testing.T) {
	app := New()
	var calls []string
	app.UseMiddleware(func(*QContext) { calls = append(calls, "root") })
	node := app.Group("/node")
	node.UseMiddleware(func(*QContext) { calls = append(calls, "node") })
	node.GET("/child", func(*QContext) { calls = append(calls, "handler") })
	performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/node/child")
	if !reflect.DeepEqual(calls, []string{"root", "node", "handler"}) {
		t.Fatalf("default calls = %v", calls)
	}
}

func TestMiddlewareClearInheritedSubtree(t *testing.T) {
	app := New()
	app.UseMiddleware(func(*QContext) { t.Fatal("cleared ancestor middleware ran") })
	public := app.Group("/public")
	endpoint := func(ctx *QContext) { ctx.SetText("ok") }
	public.GET("", endpoint)
	public.GET("/existing/child", endpoint)
	public.UseMiddleware(nil).ModeSet().Inherit(true)
	public.GET("/future/child", endpoint)
	for _, path := range []string{"/public", "/public/existing/child", "/public/future/child"} {
		response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, path)
		if response.Code != http.StatusOK || response.Body.String() != "ok" {
			t.Fatalf("%s response = (%d, %q)", path, response.Code, response.Body.String())
		}
	}
}

func TestMiddlewareClearLocalOnly(t *testing.T) {
	app := New()
	app.UseMiddleware(func(ctx *QContext) { ctx.SetHeader("X-Root", "enabled") })
	node := app.Group("/node")
	node.UseMiddleware(nil).ModeSet().Inherit(false)
	node.GET("", func(ctx *QContext) { ctx.SetText("local") })
	node.GET("/child", func(ctx *QContext) { ctx.SetText("child") })
	handler := &qHttpHandler{rootNode: app.QNode}
	local := performRequest(handler, http.MethodGet, "/node")
	child := performRequest(handler, http.MethodGet, "/node/child")
	if local.Code != http.StatusOK || local.Header().Get("X-Root") != "" {
		t.Fatalf("local response = %d, headers = %v", local.Code, local.Header())
	}
	if child.Code != http.StatusOK || child.Header().Get("X-Root") != "enabled" {
		t.Fatalf("child response = %d, headers = %v", child.Code, child.Header())
	}
}

func TestMiddlewareNextAndAbort(t *testing.T) {
	app := New()
	var calls []string
	app.UseMiddleware(func(ctx *QContext) {
		calls = append(calls, "before")
		ctx.Next()
		calls = append(calls, "after")
	}).Inherit(true)
	app.Group("/private").UseMiddleware(func(ctx *QContext) {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, TMap{"error": "denied"})
		ctx.Next()
		if !ctx.IsAborted() {
			t.Fatal("context should be aborted")
		}
	}).Inherit(true)
	app.GET("/private", func(*QContext) { t.Fatal("aborted handler ran") })
	app.GET("/public", func(ctx *QContext) {
		calls = append(calls, "handler")
		ctx.SetText("ok")
	})
	handler := &qHttpHandler{rootNode: app.QNode}
	response := performRequest(handler, http.MethodGet, "/private")
	if response.Code != http.StatusUnauthorized || response.Body.String() != `{"error":"denied"}` {
		t.Fatalf("aborted response = (%d, %q)", response.Code, response.Body.String())
	}
	if !reflect.DeepEqual(calls, []string{"before", "after"}) {
		t.Fatalf("aborted calls = %v", calls)
	}
	calls = nil
	response = performRequest(handler, http.MethodGet, "/public")
	if response.Body.String() != "ok" || !reflect.DeepEqual(calls, []string{"before", "handler", "after"}) {
		t.Fatalf("public response = %q, calls = %v", response.Body.String(), calls)
	}
}

func TestMiddlewareSetAndAdd(t *testing.T) {
	app := New()
	var calls []string
	record := func(name string) THandlerFunc {
		return func(*QContext) { calls = append(calls, name) }
	}
	app.UseMiddleware(record("discarded")).Inherit(true)
	app.UseMiddleware(nil).ModeSet().Inherit(true)
	handlers := []THandlerFunc{record("first"), nil}
	app.UseMiddleware(handlers...).Inherit(true)
	handlers[0] = record("mutated")
	app.UseMiddleware(record("second")).ModeAdd().Inherit(true)
	app.GET("/:id/*path", func(ctx *QContext) {
		calls = append(calls, ctx.Param("id", "")+"/"+ctx.Param("path", ""))
	})
	performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/42/a/b")
	if !reflect.DeepEqual(calls, []string{"first", "second", "42/a/b"}) {
		t.Fatalf("calls = %v", calls)
	}
}

func TestFileServerClearsMiddleware(t *testing.T) {
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, "asset.txt"), []byte("asset"), 0600); err != nil {
		t.Fatal(err)
	}
	app := New()
	app.UseMiddleware(func(ctx *QContext) { ctx.AbortWithStatus(http.StatusForbidden) }).Inherit(true)
	app.FileServer("/assets", directory).UseMiddleware(nil).ModeSet().Inherit(true)
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/assets/asset.txt")
	if response.Code != http.StatusOK || response.Body.String() != "asset" {
		t.Fatalf("file response = (%d, %q)", response.Code, response.Body.String())
	}
}

func TestMiddlewareConfigurationRemainsAttached(t *testing.T) {
	app := New()
	var calls []string
	record := func(name string) THandlerFunc {
		return func(*QContext) { calls = append(calls, name) }
	}
	first := app.UseMiddleware(record("first"))
	// Additional registrations must not detach previously returned configurations.
	for i := 0; i < 20; i++ {
		app.UseMiddleware(nil)
	}
	second := app.UseMiddleware(record("second"))
	if first.ModeSet().ModeAdd().Inherit(true) != first {
		t.Fatal("configuration methods must return the same QMiddleware")
	}
	second.ModeSet().Inherit(true).Inherit(false)
	app.GET("/", record("root"))
	app.GET("/child", record("child"))
	handler := &qHttpHandler{rootNode: app.QNode}
	performRequest(handler, http.MethodGet, "/")
	if !reflect.DeepEqual(calls, []string{"second", "root"}) {
		t.Fatalf("root calls = %v", calls)
	}
	calls = nil
	performRequest(handler, http.MethodGet, "/child")
	if !reflect.DeepEqual(calls, []string{"first", "child"}) {
		t.Fatalf("child calls = %v", calls)
	}
}
