package server

import (
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRenderPageWritesLayoutWithRenderedTitleAndContent(t *testing.T) {
	s := &Server{devMode: false}
	s.templates = mustTemplate(t, `
{{define "layout"}}<title>{{.Title}}</title><main>{{.Content}}</main>{{end}}
{{define "entries_title"}}  Hello  {{end}}
{{define "entries_content"}}<p>{{.Message}}</p>{{end}}
`)

	req := httptest.NewRequest("GET", "/entries", nil)
	rr := httptest.NewRecorder()

	s.renderPage(rr, req, "entries_title", "entries_content", map[string]any{"Message": "World"})

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<title>Hello</title>") {
		t.Fatalf("expected trimmed title in output, got: %s", body)
	}
	if !strings.Contains(body, "<main><p>World</p></main>") {
		t.Fatalf("expected rendered content in output, got: %s", body)
	}
}

func TestRenderTemplateMissingTemplateReturnsServerError(t *testing.T) {
	s := &Server{devMode: false}
	s.templates = mustTemplate(t, `{{define "something_else"}}x{{end}}`)

	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	s.renderTemplate(rr, req, "missing_template", nil)

	if rr.Code != 500 {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "template") {
		t.Fatalf("expected template error message, got: %s", rr.Body.String())
	}
}

func mustTemplate(t *testing.T, src string) *template.Template {
	t.Helper()
	tmpl, err := template.New("test").Parse(src)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	return tmpl
}
