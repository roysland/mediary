package server

import (
	"net/http"
	"strings"
)

type requestMeta struct {
	IsHTMX      bool
	IsJSON      bool
	IsXHR       bool
	AcceptsJSON bool
	IsAJAX      bool
}

func classifyRequest(r *http.Request) requestMeta {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	accept := strings.ToLower(r.Header.Get("Accept"))

	isJSON := strings.HasPrefix(contentType, "application/json")
	isHTMX := r.Header.Get("HX-Request") == "true"
	isXHR := r.Header.Get("X-Requested-With") == "XMLHttpRequest"
	acceptsJSON := strings.Contains(accept, "application/json")

	return requestMeta{
		IsHTMX:      isHTMX,
		IsJSON:      isJSON,
		IsXHR:       isXHR,
		AcceptsJSON: acceptsJSON,
		IsAJAX:      isJSON || isXHR || isHTMX,
	}
}
