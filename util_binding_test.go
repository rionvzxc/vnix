package vnix

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

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
