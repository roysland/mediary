package server

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"roysland.me/symptomstracker/internal/auth"
	"roysland.me/symptomstracker/internal/i18n"
)

type pageTemplate struct {
	Title             string
	Content           template.HTML
	Data              any
	Locale            string
	Theme             string
	ActiveNav         string
	BuildVersion      string
	ServiceWorker     bool
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

func (s *Server) activeTemplates(locale string) (*template.Template, error) {
	if s.devMode {
		return s.loadTemplates(locale)
	}

	if tmpl, ok := s.templatesByLocale[locale]; ok {
		return tmpl, nil
	}

	if tmpl, ok := s.templatesByLocale[i18n.DefaultLocale]; ok {
		return tmpl, nil
	}

	if locale == i18n.DefaultLocale && s.templates != nil {
		return s.templates, nil
	}

	tmpl, err := s.loadTemplates(locale)
	if err != nil {
		return nil, err
	}
	if s.templatesByLocale == nil {
		s.templatesByLocale = make(map[string]*template.Template)
	}
	s.templatesByLocale[locale] = tmpl
	if locale == i18n.DefaultLocale {
		s.templates = tmpl
	}
	return tmpl, nil

}

func (s *Server) renderPage(w http.ResponseWriter, r *http.Request, titleTemplate, contentTemplate string, data any) {
	settings := s.resolveUserSettings(r)

	tmpl, err := s.activeTemplates(settings.Language)
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

	err = tmpl.ExecuteTemplate(w, "layout", pageTemplate{
		Title: strings.TrimSpace(titleBuf.String()),
		// #nosec G203 -- contentBuf contains output from html/template execution, not raw user HTML.
		Content:           template.HTML(contentBuf.String()),
		Data:              data,
		Locale:            settings.Language,
		Theme:             settings.Theme,
		ActiveNav:         activeNavFromPath(r.URL.Path),
		BuildVersion:      s.cfg.BuildVersion,
		ServiceWorker:     s.cfg.ServiceWorkerEnabled,
		DebugTemplateData: s.devMode,
	})
	if err != nil {
		respondInternalError(w, r, err.Error())
	}
}

func (s *Server) resolveTheme(r *http.Request) string {
	settings := s.resolveUserSettings(r)
	switch settings.Theme {
	case "light", "dark", "system":
		return settings.Theme
	default:
		return defaultUserSettings().Theme
	}
}

func (s *Server) resolveUserSettings(r *http.Request) UserSettings {
	settings := defaultUserSettings()
	if s.queries == nil {
		return settings
	}

	user := auth.CurrentUser(r)
	if user == nil || user.ID <= 0 {
		return settings
	}

	loaded, err := s.loadUserSettings(r.Context(), user.ID)
	if err != nil {
		return settings
	}

	if !i18n.IsSupportedLocale(loaded.Language) {
		loaded.Language = settings.Language
	}

	switch loaded.Theme {
	case "light", "dark", "system":
	default:
		loaded.Theme = settings.Theme
	}

	return loaded
}

func (s *Server) renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data any) {
	settings := s.resolveUserSettings(r)
	tmpl, err := s.activeTemplates(settings.Language)
	if err != nil {
		respondInternalError(w, r, err.Error())
		return
	}

	err = tmpl.ExecuteTemplate(w, templateName, data)
	if err != nil {
		respondInternalError(w, r, err.Error())
	}
}
