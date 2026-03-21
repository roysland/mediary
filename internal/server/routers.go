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
	s.mux.HandleFunc("/auth", s.authPage)
	s.mux.HandleFunc("/auth/e2e/login", s.e2eLogin)
	s.mux.HandleFunc("/webauthn/register/options", s.beginRegistration)
	s.mux.HandleFunc("/webauthn/register/verify", s.finishRegistration)
	s.mux.HandleFunc("/webauthn/login/options", s.beginLogin)
	s.mux.HandleFunc("/webauthn/login/verify", s.finishLogin)
	s.mux.HandleFunc("/webauthn/passkeys/options", s.beginAddPasskey)
	s.mux.HandleFunc("/webauthn/passkeys/verify", s.finishAddPasskey)
	s.mux.HandleFunc("/auth/passkeys/register/options", s.beginAddPasskey)
	s.mux.HandleFunc("/auth/passkeys/register/verify", s.finishAddPasskey)
	s.mux.HandleFunc("/auth/device-link/create", s.createDeviceLink)
	s.mux.HandleFunc("/link", s.redeemDeviceLink)
	s.mux.HandleFunc("/auth/logout", s.logout)
	s.mux.HandleFunc("/entries", s.entries)
	s.mux.HandleFunc("/api/entries", s.entriesAPI)
	s.mux.HandleFunc("/entry/add", s.addEntry)
	s.mux.HandleFunc("/entry/voice", s.addVoiceEntry)
	s.mux.HandleFunc("/entry/{id}/delete", s.deleteEntry)
	s.mux.HandleFunc("/entry/{id}/edit", s.editEntry)
	s.mux.HandleFunc("/entry/{id}", s.entryItem)
	s.mux.HandleFunc("/entry/{id}/trackables", s.entryTrackablesDialog)
	s.mux.HandleFunc("/settings", s.settings)
	s.mux.HandleFunc("/settings/export-data", s.exportUserData)
	s.mux.HandleFunc("/settings/clear-data", s.clearUserData)
	s.mux.HandleFunc("/trackables/add", s.addTrackable)
	s.mux.HandleFunc("/trackables", s.listTrackables)
	s.mux.HandleFunc("/trackables/{id}/dismissal", s.saveTrackableDismissal)
	s.mux.HandleFunc("/trackables/{id}/delete", s.deleteTrackable)
	s.mux.HandleFunc("/trackables/{id}/add", s.saveTrackableValue)
	s.mux.HandleFunc("/trackables/{id}", s.registerTrackable)

	// Wrap all routes once so CSRF checks apply uniformly.
	s.handler = withCrossOriginProtection(withSessionRequired(s.mux), s.cfg)
}
