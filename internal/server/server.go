package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	webauthnlib "github.com/go-webauthn/webauthn/webauthn"
	_ "github.com/mattn/go-sqlite3"
	"roysland.me/symptomstracker/internal/auth"
	"roysland.me/symptomstracker/internal/db"
	"roysland.me/symptomstracker/internal/i18n"
)

type Server struct {
	mux                 *http.ServeMux
	handler             http.Handler
	templates           *template.Template
	templatesByLocale   map[string]*template.Template
	devMode             bool
	queries             *db.Queries
	dbConn              *sql.DB
	cfg                 Config
	transcriptionWorker *TranscriptionWorker
	authSessions        *auth.SessionManager
	webauthn            *webauthnlib.WebAuthn
	ceremonyMu          sync.Mutex
	ceremonies          map[string]webauthnCeremony
}

func New(cfg Config) *Server {
	conn, err := sql.Open("sqlite3", cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := conn.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := runMigrations(conn); err != nil {
		log.Fatal(err)
	}

	queries := db.New(conn)

	authSessionSecret := strings.TrimSpace(cfg.AuthSessionSecret)
	if cfg.DevMode && authSessionSecret == "" {
		authSessionSecret = "dev-only-insecure-session-secret-change-me"
	}
	if authSessionSecret == "" {
		log.Fatal("AUTH_SESSION_SECRET is required")
	}

	authSessions, err := auth.NewSessionManager(authSessionSecret, !cfg.DevMode)
	if err != nil {
		log.Fatal(err)
	}
	auth.SetDefaultSessionManager(authSessions)

	if err := validateWebAuthnConfig(cfg); err != nil {
		log.Fatalf("invalid WebAuthn configuration: %v", err)
	}

	webauthn, err := webauthnlib.New(&webauthnlib.Config{
		RPID:          cfg.WebAuthnRPID,
		RPDisplayName: cfg.WebAuthnRPDisplayName,
		RPOrigins:     cfg.WebAuthnRPOrigins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
			RequireResidentKey:      protocol.ResidentKeyRequired(),
			UserVerification:        protocol.VerificationPreferred,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	s := &Server{
		mux:               http.NewServeMux(),
		devMode:           cfg.DevMode,
		queries:           queries,
		dbConn:            conn,
		cfg:               cfg,
		templatesByLocale: make(map[string]*template.Template),
		authSessions:      authSessions,
		webauthn:          webauthn,
		ceremonies:        make(map[string]webauthnCeremony),
	}

	worker := newTranscriptionWorker(queries, cfg)
	s.transcriptionWorker = worker
	ctx := context.Background()
	worker.Start(ctx)
	worker.RecoverPending(ctx)

	if !s.devMode {
		for _, locale := range i18n.Locales() {
			tmpl, err := s.loadTemplates(locale)
			if err != nil {
				log.Fatal(err)
			}
			s.templatesByLocale[locale] = tmpl
		}
		s.templates = s.templatesByLocale[i18n.DefaultLocale]
	}

	s.routes()

	return s
}

func templateFuncMap(locale string) template.FuncMap {
	return template.FuncMap{
		"t": func(key string) string {
			return i18n.TForLocale(locale, key)
		},
		"formatUnix": func(ts int64) string {
			return time.Unix(ts, 0).UTC().Format(dateTimeLayoutUTC)
		},
		"formatISO": func(ts int64) string {
			return time.Unix(ts, 0).UTC().Format(time.RFC3339)
		},
		"json": func(v any) template.JS {
			b, err := json.Marshal(v)
			if err != nil {
				return template.JS("null")
			}
			return template.JS(b)
		},
	}
}

func (s *Server) loadTemplates(locale string) (*template.Template, error) {
	tmpl := template.New("").Funcs(templateFuncMap(locale))

	var files []string
	err := filepath.WalkDir("internal/views", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)

	return tmpl.ParseFiles(files...)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.handler == nil {
		s.handler = withCrossOriginProtection(s.mux, s.cfg)
	}
	s.handler.ServeHTTP(w, r)
}

func withCrossOriginProtection(next http.Handler, cfg Config) http.Handler {
	protection := http.NewCrossOriginProtection()
	for _, origin := range cfg.CSRFTrustedOrigins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			log.Printf("warning: invalid CSRF trusted origin %q: %v", origin, err)
		}
	}
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respondForbidden(w, r, "Forbidden")
	}))

	return protection.Handler(next)
}
