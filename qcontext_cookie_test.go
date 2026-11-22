package vnix

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestContextReadsCookies(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "xyz"})
	request.AddCookie(&http.Cookie{Name: "session", Value: "123"})
	ctx := newContext(request, httptest.NewRecorder(), nil)
	if ctx.Cookie("session", "") != "xyz" || strings.Join(ctx.CookieValues("session"), ",") != "xyz,123" {
		t.Fatal("cookie accessors returned unexpected values")
	}
	if ctx.Cookie("missing", "") != "" || ctx.Cookie("missing", "fallback") != "fallback" || ctx.CookieValues("missing") != nil {
		t.Fatal("missing cookie defaults are incorrect")
	}
}

func TestContextCookieAccessorsReturnCopies(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "first"})
	request.AddCookie(&http.Cookie{Name: "session", Value: "second"})
	ctx := newContext(request, httptest.NewRecorder(), nil)
	values := ctx.CookieValues("session")
	values[0] = "changed"
	cookies := ctx.RequestCookie()
	cookies["session"][0] = "changed"
	cookies["new"] = []string{"value"}
	if !reflect.DeepEqual(ctx.CookieValues("session"), []string{"first", "second"}) || ctx.reqCookie.Exist("new") {
		t.Fatal("cookie accessors mutated the original cookies")
	}
}

func TestContextSetCookie(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetCookie([]*TCookie{{Name: "session", Value: "active"}, nil})
		ctx.SetText("ok")
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "session" || cookies[0].Value != "active" {
		t.Fatalf("response cookies = %#v", cookies)
	}
}

func TestContextAddCookie(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetCookie([]*TCookie{{Name: "session", Value: "active", Path: "/"}})
		ctx.AddCookie([]*TCookie{{Name: "theme", Value: "dark"}, nil})
		ctx.AddCookie([]*TCookie{{Name: "session", Value: "admin", Path: "/admin"}})
		ctx.AddCookie(nil)
		ctx.AddCookie([]*TCookie{})
		ctx.SetText("ok")
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	cookies := response.Result().Cookies()
	if len(cookies) != 3 {
		t.Fatalf("cookie count = %d, want 3", len(cookies))
	}
	if cookies[0].Name != "session" || cookies[0].Value != "active" || cookies[0].Path != "/" {
		t.Fatalf("existing cookie was changed: %#v", cookies[0])
	}
	if cookies[1].Name != "theme" || cookies[1].Value != "dark" {
		t.Fatalf("added cookie = %#v", cookies[1])
	}
	if cookies[2].Name != "session" || cookies[2].Value != "admin" || cookies[2].Path != "/admin" {
		t.Fatalf("same-name cookie on another path = %#v", cookies[2])
	}
}

func TestContextSetCookieReplacesAddedCookies(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.AddCookie([]*TCookie{{Name: "old", Value: "value"}})
		ctx.SetCookie([]*TCookie{{Name: "new", Value: "value"}})
		ctx.SetText("ok")
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/")
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "new" {
		t.Fatalf("replaced cookies = %#v", cookies)
	}
}

func TestContextSetCookieCopiesSlice(t *testing.T) {
	cookies := make([]*TCookie, 1, 4)
	cookies[0] = &TCookie{Name: "theme", Value: "dark"}
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), nil)
	ctx.SetCookie(cookies)
	cookies[0] = &TCookie{Name: "replaced", Value: "outside"}
	ctx.AddCookie([]*TCookie{{Name: "session", Value: "private"}})
	if ctx.resCookies[0].Name != "theme" || cookies[:cap(cookies)][1] != nil {
		t.Fatal("response cookie list shares storage with the caller")
	}
	ctx.SetCookie(nil)
	if len(ctx.resCookies) != 0 {
		t.Fatal("SetCookie(nil) must clear the response cookie list")
	}
}

func TestContextCookiesAreIsolatedAcrossRequests(t *testing.T) {
	defaults := make([]*TCookie, 1, 2)
	defaults[0] = &TCookie{Name: "theme", Value: "dark"}
	aReady := make(chan struct{})
	bDone := make(chan struct{})
	app := New()
	app.UseMiddleware(func(ctx *QContext) { ctx.SetCookie(defaults) })
	app.GET("/a", func(ctx *QContext) {
		ctx.AddCookie([]*TCookie{{Name: "session", Value: "user-a"}})
		close(aReady)
		select {
		case <-bDone:
		case <-time.After(5 * time.Second):
			t.Error("request B did not complete its handler")
		}
		ctx.SetText("a")
	})
	app.GET("/b", func(ctx *QContext) {
		ctx.AddCookie([]*TCookie{{Name: "session", Value: "user-b"}})
		close(bDone)
		ctx.SetText("b")
	})
	server := httptest.NewServer(&qHttpHandler{rootNode: app.QNode})
	defer server.Close()
	type result struct {
		response *http.Response
		err      error
	}
	aResult := make(chan result, 1)
	go func() {
		response, err := server.Client().Get(server.URL + "/a")
		aResult <- result{response, err}
	}()
	select {
	case <-aReady:
	case <-time.After(5 * time.Second):
		t.Fatal("request A did not start")
	}
	bResponse, err := server.Client().Get(server.URL + "/b")
	if err != nil {
		t.Fatal(err)
	}
	defer bResponse.Body.Close()
	a := <-aResult
	if a.err != nil {
		t.Fatal(a.err)
	}
	defer a.response.Body.Close()
	for _, expected := range []struct {
		response *http.Response
		session  string
	}{{a.response, "user-a"}, {bResponse, "user-b"}} {
		cookies := expected.response.Cookies()
		if len(cookies) != 2 || cookies[0].Value != "dark" || cookies[1].Name != "session" || cookies[1].Value != expected.session {
			t.Fatalf("expected session %q, received %#v", expected.session, cookies)
		}
	}
}

func TestContextBindCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(&http.Cookie{Name: "session", Value: "active"})
	ctx := newContext(request, httptest.NewRecorder(), nil)
	var cookies struct {
		Session string `cookie:"session"`
	}
	if err := ctx.BindCookie(&cookies); err != nil || cookies.Session != "active" {
		t.Fatalf("BindCookie() = (%#v, %v)", cookies, err)
	}
}
