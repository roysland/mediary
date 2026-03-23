package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"roysland.me/symptomstracker/internal/server"
)

func main() {
	loadDevDotEnv()

	cfg := server.LoadConfig()

	s := server.New(cfg)

	log.Printf("Starting on %s\n", cfg.ListenAddr)

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           s,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	err := httpServer.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func loadDevDotEnv() {
	if os.Getenv("APP_ENV") == "production" {
		return
	}

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("warning: unable to load .env: %v", err)
	}
}
