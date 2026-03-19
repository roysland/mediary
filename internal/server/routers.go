package server

import (
	"net/http"
)

func (s *Server) routes() {
	fs := http.FileServer(http.Dir("web/static"))
	s.mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve audio files from the audio storage directory.
	audioFS := http.FileServer(http.Dir(s.cfg.AudioStorageDir))
	s.mux.Handle("/data/audio/", http.StripPrefix("/data/audio/", audioFS))

	s.mux.HandleFunc("/", s.home)
	s.mux.HandleFunc("/entries", s.entries)
	s.mux.HandleFunc("/api/entries", s.entriesAPI)
	s.mux.HandleFunc("/entry/add", s.addEntry)
	s.mux.HandleFunc("/entry/voice", s.addVoiceEntry)
	s.mux.HandleFunc("/entry/{id}/delete", s.deleteEntry)
	s.mux.HandleFunc("/entry/{id}", s.entryItem)
	s.mux.HandleFunc("/entry/{id}/trackables", s.entryTrackablesDialog)
	s.mux.HandleFunc("/settings", s.settings)
	s.mux.HandleFunc("/settings/export-data", s.exportUserData)
	s.mux.HandleFunc("/settings/clear-data", s.clearUserData)
	s.mux.HandleFunc("/trackables/add", s.addTrackable)
	s.mux.HandleFunc("/trackables", s.listTrackables)
	s.mux.HandleFunc("/trackables/{id}/dismissal", s.saveTrackableDismissal)
	s.mux.HandleFunc("/trackables/{id}/add", s.saveTrackableValue)
	s.mux.HandleFunc("/trackables/{id}", s.registerTrackable)

	// Wrap all routes once so CSRF checks apply uniformly.
	s.handler = withCrossOriginProtection(s.mux, s.cfg)
}
