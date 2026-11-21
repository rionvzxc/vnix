package vnix

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestContextParamValues(t *testing.T) {
	app := New()
	app.GET("/items/:id/:id", func(ctx *QContext) {
		ctx.SetText(strings.Join([]string{
			ctx.Param("id", ""),
			strings.Join(ctx.ParamValues("id"), ","),
			ctx.Param("missing", "default"),
		}, ":"))
	})
	response := performRequest(&qHttpHandler{rootNode: app.QNode}, http.MethodGet, "/items/1/2")

	if response.Code != http.StatusOK || response.Body.String() != "1:1,2:default" {
		t.Fatalf("parameter response = (%d, %q)", response.Code, response.Body.String())
	}
}
func TestContextParamAccessorsReturnCopies(t *testing.T) {
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), TValues{"id": {"first", "second"}})
	values := ctx.ParamValues("id")
	values[0] = "changed"
	params := ctx.RequestParam()
	params["id"][0] = "changed"
	params["new"] = []string{"value"}
	if !reflect.DeepEqual(ctx.ParamValues("id"), []string{"first", "second"}) || ctx.reqParam.Exist("new") {
		t.Fatal("parameter accessors mutated the original values")
	}
}

func TestContextMissingParams(t *testing.T) {
	ctx := newContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder(), nil)
	if ctx.Param("missing", "") != "" || ctx.Param("missing", "fallback") != "fallback" || ctx.ParamValues("missing") != nil {
		t.Fatal("missing parameter defaults are incorrect")
	}
}

func TestContextBindParam(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	ctx := newContext(request, httptest.NewRecorder(), TValues{"id": {"42"}})
	var params struct {
		ID int `uri:"id"`
	}
	if err := ctx.BindParam(&params); err != nil || params.ID != 42 {
		t.Fatalf("BindParam() = (%d, %v), want (42, nil)", params.ID, err)
	}
}
