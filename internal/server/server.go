package server

import (
	"database/sql"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"roysland.me/symptomstracker/internal/db"
	"roysland.me/symptomstracker/internal/i18n"
)

type Server struct {
	mux       *http.ServeMux
	templates *template.Template
	devMode   bool
	queries   *db.Queries
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

	s := &Server{
		mux:     http.NewServeMux(),
		devMode: cfg.DevMode,
		queries: queries,
	}

	if !s.devMode {
		tmpl, err := s.loadTemplates()
		if err != nil {
			log.Fatal(err)
		}
		s.templates = tmpl
	}

	s.routes()

	return s
}

func (s *Server) loadTemplates() (*template.Template, error) {
	tmpl := template.New("").Funcs(template.FuncMap{
		"t": i18n.T,
		"formatUnix": func(ts int64) string {
			return time.Unix(ts, 0).UTC().Format("2006-01-02 15:04:05")
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
	})

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
	s.mux.ServeHTTP(w, r)
}
