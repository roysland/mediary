package server

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"roysland.me/symptomstracker/internal/auth"
)

type pageTemplate struct {
	Title             string
	Content           template.HTML
	Data              any
	Theme             string
	ActiveNav         string
	DebugTemplateData bool
}

func activeNavFromPath(path string) string {
	switch {
	case path == "/":
		return "home"
	case path == "/entries" || strings.HasPrefix(path, "/entry/"):
		return "entries"
	case strings.HasPrefix(path, "/trackables"):
		return "trackables"
	case strings.HasPrefix(path, "/settings"):
		return "settings"
	default:
		return ""
	}
}

func (s *Server) activeTemplates() (*template.Template, error) {
	tmpl := s.templates
	if s.devMode {
		var err error
		tmpl, err = s.loadTemplates()
		if err != nil {
			return nil, err
		}
	}
	return tmpl, nil
}

func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, titleTemplate, contentTemplate string, data any) {
	tmpl, err := s.activeTemplates()
	if err != nil {
		respondInternalError(w, r, err.Error())
		return
	}

	var titleBuf bytes.Buffer
	err = tmpl.ExecuteTemplate(&titleBuf, titleTemplate, data)
	if err != nil {
		respondInternalError(w, r, err.Error())
		return
	}

	var contentBuf bytes.Buffer
	err = tmpl.ExecuteTemplate(&contentBuf, contentTemplate, data)
	if err != nil {
		respondInternalError(w, r, err.Error())
		return
	}

	theme := s.resolveTheme(r)

	err = tmpl.ExecuteTemplate(w, "layout", pageTemplate{
		Title:             strings.TrimSpace(titleBuf.String()),
		Content:           template.HTML(contentBuf.String()),
		Data:              data,
		Theme:             theme,
		ActiveNav:         activeNavFromPath(r.URL.Path),
		DebugTemplateData: s.devMode,
	})
	if err != nil {
		respondInternalError(w, r, err.Error())
	}
}

func (s *Server) resolveTheme(r *http.Request) string {
	settings := defaultUserSettings()
	if s.queries == nil {
		return settings.Theme
	}

	user := auth.CurrentUser(r)
	if user == nil || user.ID <= 0 {
		return settings.Theme
	}

	loaded, err := s.loadUserSettings(r.Context(), user.ID)
	if err != nil {
		return settings.Theme
	}

	switch loaded.Theme {
	case "light", "dark", "system":
		return loaded.Theme
	default:
		return settings.Theme
	}
}

func (s *Server) renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data any) {
	tmpl, err := s.activeTemplates()
	if err != nil {
		respondInternalError(w, r, err.Error())
		return
	}

	err = tmpl.ExecuteTemplate(w, templateName, data)
	if err != nil {
		respondInternalError(w, r, err.Error())
	}
}
