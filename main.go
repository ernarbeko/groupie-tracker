package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/handlers"
)

func main() {
	addr := envOr("ADDR", ":8080")
	baseURL := envOr("API_URL", "https://groupietrackers.herokuapp.com/api")

	client := api.NewClient(baseURL)
	if err := client.Refresh(); err != nil {
		log.Printf("warning: initial API load failed: %v (will retry on request)", err)
	}

	srv, err := handlers.New(client, "templates")
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/", srv.Home)
	mux.HandleFunc("/artist/", srv.Artist)
	mux.HandleFunc("/api/search", srv.SearchAPI)

	server := &http.Server{
		Addr:              addr,
		Handler:           handlers.Recover(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("Groupie Tracker listening on http://localhost%s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
