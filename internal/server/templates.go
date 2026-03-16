package server

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"
)

type pageTemplate struct {
	Title   string
	Content template.HTML
	Data    any
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

	err = tmpl.ExecuteTemplate(w, "layout", pageTemplate{
		Title:   strings.TrimSpace(titleBuf.String()),
		Content: template.HTML(contentBuf.String()),
		Data:    data,
	})
	if err != nil {
		respondInternalError(w, r, err.Error())
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
