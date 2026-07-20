package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"groupie-tracker/internal/api"
	"groupie-tracker/internal/models"
)

func testServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()

	artists := []models.Artist{{
		ID:           1,
		Name:         "Queen",
		Image:        "https://example.com/queen.jpg",
		Members:      []string{"Freddie Mercury"},
		CreationDate: 1970,
		FirstAlbum:   "14-12-1973",
	}}
	relations := models.RelationsResponse{
		Index: []models.RelationEntry{{
			ID:             1,
			DatesLocations: map[string][]string{"london-uk": {"01-01-2020"}},
		}},
	}
	locations := models.LocationsResponse{
		Index: []models.LocationEntry{
			{ID: 1, Locations: []string{"london-uk"}},
		},
	}
	dates := models.DatesResponse{
		Index: []models.DateEntry{
			{ID: 1, Dates: []string{"01-01-2020"}},
		},
	}

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/artists":
			_ = json.NewEncoder(w).Encode(artists)
		case "/relation":
			_ = json.NewEncoder(w).Encode(relations)
		case "/locations":
			_ = json.NewEncoder(w).Encode(locations)
		case "/dates":
			_ = json.NewEncoder(w).Encode(dates)
		default:
			http.NotFound(w, r)
		}
	}))

	root, err := filepath.Abs(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("templates dir: %v", err)
	}

	client := api.NewClient(apiServer.URL)
	srv, err := New(client, root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return srv, apiServer
}

func TestHomeOK(t *testing.T) {
	srv, apiServer := testServer(t)
	defer apiServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	srv.Home(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Queen") {
		t.Fatalf("body missing artist: %q", body)
	}
}

func TestHomeNotFound(t *testing.T) {
	srv, apiServer := testServer(t)
	defer apiServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/nope", nil)
	rec := httptest.NewRecorder()
	srv.Home(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestSearchAPI(t *testing.T) {
	srv, apiServer := testServer(t)
	defer apiServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/search?q=queen", nil)
	rec := httptest.NewRecorder()
	srv.SearchAPI(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	var payload struct {
		Count   int                `json:"count"`
		Results []models.SearchHit `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v", err)
	}
	if payload.Count != 1 || payload.Results[0].Name != "Queen" {
		t.Fatalf("payload = %+v", payload)
	}
}

func TestArtistOK(t *testing.T) {
	srv, apiServer := testServer(t)
	defer apiServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/artist/1", nil)
	rec := httptest.NewRecorder()
	srv.Artist(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
