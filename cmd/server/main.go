package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"roysland.me/symptomstracker/internal/server"
)

func main() {
	loadDevDotEnv()

	cfg := server.LoadConfig()

	s := server.New(cfg)

	log.Printf("Starting on %s\n", cfg.ListenAddr)

	err := http.ListenAndServe(cfg.ListenAddr, s)
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
