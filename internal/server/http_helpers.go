package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

func requireMethod(w http.ResponseWriter, r *http.Request, allowed ...string) bool {
	for _, method := range allowed {
		if r.Method == method {
			return true
		}
	}

	w.Header().Set("Allow", strings.Join(allowed, ", "))
	respondMethodNotAllowed(w, r)
	return false
}

func requirePathInt64(w http.ResponseWriter, r *http.Request, paramName, label string) (int64, bool) {
	raw := strings.TrimSpace(r.PathValue(paramName))
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		respondBadRequest(w, r, fmt.Sprintf("Invalid %s", label))
		return 0, false
	}

	return value, true
}

func requireParsedForm(w http.ResponseWriter, r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		respondBadRequest(w, r, "Invalid form data")
		return false
	}

	return true
}
