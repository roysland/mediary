package server

import (
	"encoding/json"
	"net/http"
)

func respondJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

func respondError(w http.ResponseWriter, r *http.Request, statusCode int, message string) {
	if r != nil {
		req := classifyRequest(r)
		if req.IsAJAX || req.AcceptsJSON {
			respondJSON(w, statusCode, map[string]any{
				"status": "error",
				"error":  message,
			})
			return
		}
	}

	http.Error(w, message, statusCode)
}

func respondMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusMethodNotAllowed, "Method not allowed")
}

func respondBadRequest(w http.ResponseWriter, r *http.Request, message string) {
	respondError(w, r, http.StatusBadRequest, message)
}

func respondUnauthorized(w http.ResponseWriter, r *http.Request) {
	respondError(w, r, http.StatusUnauthorized, "Unauthorized")
}

func respondForbidden(w http.ResponseWriter, r *http.Request, message string) {
	respondError(w, r, http.StatusForbidden, message)
}

func respondNotFound(w http.ResponseWriter, r *http.Request, message string) {
	respondError(w, r, http.StatusNotFound, message)
}

func respondInternalError(w http.ResponseWriter, r *http.Request, message string) {
	respondError(w, r, http.StatusInternalServerError, message)
}
