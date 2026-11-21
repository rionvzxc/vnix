package vnix

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
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
