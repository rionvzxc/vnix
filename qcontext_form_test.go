package vnix

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFormParseErrorPersists(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{"urlencoded", "application/x-www-form-urlencoded", "name=valid&broken=%ZZ"},
		{"multipart", "multipart/form-data; boundary=test", "--test\r\nInvalid-Header\r\n\r\nvalue\r\n--test--\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", tt.contentType)
			ctx := newContext(request, httptest.NewRecorder(), nil)
			var input struct {
				Name string `form:"name"`
			}
			firstErr := ctx.BindForm(&input)
			if firstErr == nil {
				t.Fatal("BindForm() should reject malformed form data")
			}
			if err := ctx.BindForm(&input); err != firstErr {
				t.Fatalf("second error = %v, want %v", err, firstErr)
			}
			if got := ctx.Form("name", "fallback"); got != "fallback" {
				t.Fatalf("Form() = %q, want fallback", got)
			}
			if ctx.FormValues("name") != nil || ctx.RequestForm() != nil {
				t.Fatal("form accessors should reject partially parsed data")
			}
			if _, err := ctx.Files("upload"); err != firstErr {
				t.Fatalf("Files() error = %v, want %v", err, firstErr)
			}
		})
	}
}

func TestFormReadsURLEncodedData(t *testing.T) {
	form := url.Values{"id": {"42"}, "name": {"vnix"}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	app := New()
	app.POST("/", func(ctx *QContext) {
		id := ctx.Form("id", "")
		name := ctx.Form("name", "Unknown")
		ctx.SetText(id + ":" + name)
	})
	(&qHttpHandler{rootNode: app.QNode}).ServeHTTP(response, request)

	if response.Code != http.StatusOK || response.Body.String() != "42:vnix" {
		t.Fatalf("form response = (%d, %q)", response.Code, response.Body.String())
	}
}

func TestFormAndForms(t *testing.T) {
	form := url.Values{"tag": {"first", "second"}, "empty": {""}}
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := newContext(request, httptest.NewRecorder(), nil)

	if got := ctx.Form("missing", "fallback"); got != "fallback" {
		t.Fatalf("Form() for missing key = %q", got)
	}
	if got := ctx.Form("empty", "fallback"); got != "" {
		t.Fatalf("Form() for present empty value = %q", got)
	}
	if got := ctx.Form("tag", ""); got != "first" {
		t.Fatalf("Form() = %q, want first", got)
	}
	values := ctx.FormValues("tag")

	if len(values) != 2 || values[0] != "first" || values[1] != "second" {
		t.Fatalf("FormValues() = %#v", values)
	}
	values[0] = "changed"
	if got := ctx.Form("tag", ""); got != "first" {
		t.Fatalf("mutating FormValues() result changed stored form value to %q", got)
	}
	allValues := ctx.RequestForm()
	allValues["tag"][0] = "changed"
	if got := ctx.Form("tag", ""); got != "first" {
		t.Fatalf("mutating RequestForm() result changed stored form value to %q", got)
	}
}

func TestFormAfterRawBodyReadReturnsDefault(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("name=value"))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	ctx := newContext(request, httptest.NewRecorder(), nil)

	if _, err := ctx.Data(); err != nil {
		t.Fatal(err)
	}
	if got := ctx.Form("name", "fallback"); got != "fallback" {
		t.Fatalf("Form() after Data() = %q, want fallback", got)
	}
}

func TestFormHandlesMalformedMultipart(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("malformed"))
	request.Header.Set("Content-Type", "multipart/form-data; boundary=missing")
	ctx := newContext(request, httptest.NewRecorder(), nil)
	if got := ctx.FormValues("name"); got != nil {
		t.Fatalf("FormValues() for malformed multipart = %#v", got)
	}
	if got := ctx.Form("name", "fallback"); got != "fallback" {
		t.Fatalf("Form() for malformed multipart = %q", got)
	}
}
func TestContextBindForm(t *testing.T) {
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
}
