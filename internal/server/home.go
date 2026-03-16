package server

import (
	"net/http"
)

func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	s.renderPage(w, r, "home_title", "home_content", nil)
}
