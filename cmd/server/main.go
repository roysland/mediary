package main

import (
	"log"
	"net/http"

	"roysland.me/symptomstracker/internal/server"
)

func main() {
	cfg := server.LoadConfig()

	s := server.New(cfg)

	log.Printf("Starting on %s\n", cfg.ListenAddr)

	err := http.ListenAndServe(cfg.ListenAddr, s)
	if err != nil {
		log.Fatal(err)
	}
}
