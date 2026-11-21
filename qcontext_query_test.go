package vnix

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestContextReadsQuery(t *testing.T) {
	app := New()
	app.GET("/", func(ctx *QContext) {
		ctx.SetText(strings.Join([]string{
			ctx.Query("tag", ""),
			strings.Join(ctx.QueryValues("tag"), ","),
			ctx.Query("missing", "default"),
		}, ":"))
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/?tag=go&tag=vnix")
	if response.Code != http.StatusOK || response.Body.String() != "go:go,vnix:default" {
		t.Fatalf("query response = (%d, %q)", response.Code, response.Body.String())
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
func TestContextQueryAccessorsReturnCopies(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?tag=first&tag=second", nil)
	ctx := newContext(request, httptest.NewRecorder(), nil)
	values := ctx.QueryValues("tag")
	values[0] = "changed"
	query := ctx.RequestQuery()
	query["tag"][0] = "changed"
	query["new"] = []string{"value"}
	if !reflect.DeepEqual(ctx.QueryValues("tag"), []string{"first", "second"}) || ctx.reqQuery.Exist("new") {
		t.Fatal("query accessors mutated the original values")
	}
}

func TestContextMissingQuery(t *testing.T) {
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), nil)
	if ctx.Query("missing", "") != "" || ctx.Query("missing", "fallback") != "fallback" || ctx.QueryValues("missing") != nil {
		t.Fatal("missing query defaults are incorrect")
	}
}

func TestContextBindQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?name=vnix&tag=go&tag=http", nil)
	ctx := newContext(request, httptest.NewRecorder(), nil)
	var query struct {
		Name string   `query:"name"`
		Tags []string `query:"tag"`
	}
	if err := ctx.BindQuery(&query); err != nil || query.Name != "vnix" || strings.Join(query.Tags, ",") != "go,http" {
		t.Fatalf("BindQuery() = (%#v, %v)", query, err)
	}
}
