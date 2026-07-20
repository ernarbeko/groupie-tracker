package handlers

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"groupie-tracker/internal/api"
)

// Server holds dependencies for HTTP handlers.
type Server struct {
	api       *api.Client
	templates *template.Template
}

// New creates a Server with parsed templates from dir.
func New(client *api.Client, templatesDir string) (*Server, error) {
	pattern := filepath.Join(templatesDir, "*.html")
	tmpl, err := template.New("").Funcs(template.FuncMap{
		"join": strings.Join,
		"formatLocation": func(raw string) string {
			return api.FormatLocation(raw)
		},
	}).ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	return &Server{api: client, templates: tmpl}, nil
}

type pageData struct {
	Title   string
	Artists any
	Artist  any
	Query   string
	Error   string
	Status  int
}

// Home renders the artist gallery.
func (s *Server) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		s.renderError(w, http.StatusNotFound, "Page not found")
		return
	}
	if r.Method != http.MethodGet {
		s.renderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	artists, err := s.api.Artists()
	if err != nil {
		log.Printf("home: load artists: %v", err)
		s.renderError(w, http.StatusBadGateway, "Unable to load artists right now")
		return
	}
	s.render(w, "index.html", http.StatusOK, pageData{
		Title:   "Groupie Tracker",
		Artists: artists,
	})
}

// Artist renders a single artist detail page.
func (s *Server) Artist(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.renderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/artist/")
	idStr = strings.Trim(idStr, "/")
	if idStr == "" || strings.Contains(idStr, "/") {
		s.renderError(w, http.StatusNotFound, "Artist not found")
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		s.renderError(w, http.StatusNotFound, "Artist not found")
		return
	}

	detail, err := s.api.ArtistDetail(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.renderError(w, http.StatusNotFound, "Artist not found")
			return
		}
		log.Printf("artist %d: %v", id, err)
		s.renderError(w, http.StatusBadGateway, "Unable to load artist details")
		return
	}

	s.render(w, "artist.html", http.StatusOK, pageData{
		Title:  detail.Name + " · Groupie Tracker",
		Artist: detail,
	})
}

// SearchAPI handles the client-server search event and returns JSON.
func (s *Server) SearchAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	query := r.URL.Query().Get("q")
	hits, err := s.api.Search(query)
	if err != nil {
		log.Printf("search: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "search unavailable"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   query,
		"count":   len(hits),
		"results": hits,
	})
}

func (s *Server) render(w http.ResponseWriter, name string, status int, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s: %v", name, err)
	}
}

func (s *Server) renderError(w http.ResponseWriter, status int, message string) {
	s.render(w, "error.html", status, pageData{
		Title:  "Error · Groupie Tracker",
		Error:  message,
		Status: status,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("json encode: %v", err)
	}
}

// Recover wraps a handler so panics become 500 responses instead of crashing.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("panic recovered: %v", rec)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
