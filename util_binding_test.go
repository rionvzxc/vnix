package vnix

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestBindParamQueryHeaderAndCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/items/42?name=vnix&tag=go&tag=http", nil)
	request.Header.Add("X-Request-ID", "first")
	request.Header.Add("X-Request-ID", "second")
	request.AddCookie(&http.Cookie{Name: "session", Value: "active"})
	ctx := newContext(request, httptest.NewRecorder(), TValues{"id": {"42"}})

	var params struct {
		ID int `uri:"id"`
	}
	if err := ctx.BindParam(&params); err != nil || params.ID != 42 {
		t.Fatalf("BindParam() = (%d, %v), want (42, nil)", params.ID, err)
	}
	var query struct {
		Name string   `query:"name"`
		Tags []string `query:"tag"`
	}
	if err := ctx.BindQuery(&query); err != nil || query.Name != "vnix" || strings.Join(query.Tags, ",") != "go,http" {
		t.Fatalf("BindQuery() = (%#v, %v)", query, err)
	}
	var headers struct {
		RequestID []string `header:"X-Request-ID"`
	}
	if err := ctx.BindHeader(&headers); err != nil || strings.Join(headers.RequestID, ",") != "first,second" {
		t.Fatalf("BindHeader() = (%#v, %v)", headers, err)
	}
	var cookies struct {
		Session string `cookie:"session"`
	}
	if err := ctx.BindCookie(&cookies); err != nil || cookies.Session != "active" {
		t.Fatalf("BindCookie() = (%#v, %v)", cookies, err)
	}
}
func TestBindFormAndInvalidBindingInputs(t *testing.T) {
	form := url.Values{"term": {"go"}, "limit": {"3"}, "tag": {"api", "http"}}
	request := httptest.NewRequest(http.MethodPost, "/search", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := newContext(request, httptest.NewRecorder(), nil)
	var input struct {
		Term  string   `form:"term"`
		Limit int      `form:"limit"`
		Tags  []string `form:"tag"`
	}
	if err := ctx.BindForm(&input); err != nil || input.Term != "go" || input.Limit != 3 || strings.Join(input.Tags, ",") != "api,http" {
		t.Fatalf("BindForm() = (%#v, %v)", input, err)
	}
	if err := bindValues(input, "form", TValues{"term": {"go"}}); err == nil {
		t.Fatal("binding a non-pointer target should fail")
	}
	if err := bindValues(&struct {
		Limit int `query:"limit"`
	}{}, "query", TValues{"limit": {"invalid"}}); err == nil {
		t.Fatal("binding an invalid integer should fail")
	}
}
func TestBindQueryReturnsMalformedQueryError(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?name=vnix", nil)
	request.URL.RawQuery = "name=%zz"
	ctx := newContext(request, httptest.NewRecorder(), nil)
	if err := ctx.BindQuery(&struct {
		Name string `query:"name"`
	}{}); err == nil {
		t.Fatal("BindQuery() should return an error for malformed query values")
	}
}
