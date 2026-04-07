package server

import (
	"net/http"
)

func (s *Server) routes() {
	fs := http.FileServer(http.Dir("web/static"))
	staticHandler := http.StripPrefix("/static/", fs)
	s.mux.Handle("/static/sw.js", withServiceWorkerAllowed(staticHandler))
	s.mux.Handle("/static/", staticHandler)

	// Serve audio files from the audio storage directory.
	audioFS := http.FileServer(http.Dir(s.cfg.AudioStorageDir))
	s.mux.Handle("/data/audio/", http.StripPrefix("/data/audio/", audioFS))

	// Serve uploaded entry images from the image storage directory.
	imageFS := http.FileServer(http.Dir(s.cfg.ImageStorageDir))
	s.mux.Handle("/data/images/", http.StripPrefix("/data/images/", imageFS))

	s.mux.HandleFunc("/healthz", s.health)
	s.mux.HandleFunc("/{$}", s.home)
	s.mux.HandleFunc("/auth", s.authPage)
	s.mux.HandleFunc("/onboarding/preview", s.onboardingPreview)
	s.mux.HandleFunc("/onboarding", s.onboardingRoot)
	s.mux.HandleFunc("/onboarding/{step}/skip", s.onboardingStepPost)
	s.mux.HandleFunc("/onboarding/{step}", s.onboardingStepRoute)
	s.registerE2ERoutes()
	s.mux.HandleFunc("/webauthn/register/options", s.beginRegistration)
	s.mux.HandleFunc("/webauthn/register/verify", s.finishRegistration)
	s.mux.HandleFunc("/webauthn/login/options", s.beginLogin)
	s.mux.HandleFunc("/webauthn/login/verify", s.finishLogin)
	s.mux.HandleFunc("/webauthn/passkeys/options", s.beginAddPasskey)
	s.mux.HandleFunc("/webauthn/passkeys/verify", s.finishAddPasskey)
	s.mux.HandleFunc("/auth/passkeys/register/options", s.beginAddPasskey)
	s.mux.HandleFunc("/auth/passkeys/register/verify", s.finishAddPasskey)
	s.mux.HandleFunc("/auth/device-link/create", s.createDeviceLink)
	s.mux.HandleFunc("/share/create", withShareSecurityHeaders(s.createShareLink))
	s.mux.HandleFunc("/share/{token}", withShareSecurityHeaders(s.shareTokenRoute))
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
	s.mux.HandleFunc("/entry/{id}/images", s.uploadEntryImage)
	s.mux.HandleFunc("/entry/{id}/images/{imgID}", s.deleteEntryImage)
	s.mux.HandleFunc("/settings", s.settings)
	s.mux.HandleFunc("/settings/sensitive-content", s.settingsSensitiveContent)
	s.mux.HandleFunc("/settings/export-data", s.exportUserData)
	s.mux.HandleFunc("/settings/clear-data", s.clearUserData)
	s.mux.HandleFunc("/settings/shares", s.listShareTokens)
	s.mux.HandleFunc("/settings/shares/{id}/revoke", s.revokeShareTokenByID)
	s.mux.HandleFunc("/trackables/add", s.addTrackable)
	s.mux.HandleFunc("/trackables", s.listTrackables)
	s.mux.HandleFunc("/trackables/{id}/dismissal", s.saveTrackableDismissal)
	s.mux.HandleFunc("/trackables/{id}/delete", s.deleteTrackable)
	s.mux.HandleFunc("/trackables/{id}/add", s.saveTrackableValue)
	s.mux.HandleFunc("/trackables/{id}", s.registerTrackable)
	s.mux.HandleFunc("/alert/{version}/dismiss", s.dismissAlert)

	// Wrap all routes once so CSRF checks apply uniformly.
	s.handler = withCrossOriginProtection(withSessionRequired(s, s.mux), s.cfg)
}

func withShareSecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Robots-Tag", "noindex")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next(w, r)
	}
}

func withServiceWorkerAllowed(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Service-Worker-Allowed", "/")
		next.ServeHTTP(w, r)
	})
}
