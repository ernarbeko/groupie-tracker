package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"groupie-tracker/internal/models"
)

func TestSplitAndFormatLocation(t *testing.T) {
	city, country := SplitLocation("los_angeles-usa")
	if city != "Los Angeles" || country != "Usa" {
		t.Fatalf("SplitLocation = %q, %q", city, country)
	}
	got := FormatLocation("north_carolina-usa")
	if got != "North Carolina, Usa" {
		t.Fatalf("FormatLocation = %q", got)
	}
}

func TestTitleWords(t *testing.T) {
	if got := TitleWords("new zealand"); got != "New Zealand" {
		t.Fatalf("TitleWords = %q", got)
	}
}

func TestSearchAndArtistDetail(t *testing.T) {
	artists := []models.Artist{
		{
			ID:           1,
			Name:         "Queen",
			Image:        "https://example.com/queen.jpg",
			Members:      []string{"Freddie Mercury", "Brian May"},
			CreationDate: 1970,
			FirstAlbum:   "14-12-1973",
		},
		{
			ID:           2,
			Name:         "Pink Floyd",
			Image:        "https://example.com/pf.jpg",
			Members:      []string{"David Gilmour"},
			CreationDate: 1965,
			FirstAlbum:   "05-08-1967",
		},
	}
	relations := models.RelationsResponse{
		Index: []models.RelationEntry{
			{
				ID: 1,
				DatesLocations: map[string][]string{
					"london-uk": {"*10-02-2020"},
				},
			},
		},
	}
	locations := models.LocationsResponse{
		Index: []models.LocationEntry{
			{ID: 1, Locations: []string{"london-uk"}},
		},
	}
	dates := models.DatesResponse{
		Index: []models.DateEntry{
			{ID: 1, Dates: []string{"*10-02-2020"}},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer server.Close()

	client := NewClient(server.URL)
	hits, err := client.Search("freddie")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(hits) != 1 || hits[0].Name != "Queen" || hits[0].Match != "member" {
		t.Fatalf("unexpected member hits: %+v", hits)
	}

	hits, err = client.Search("london")
	if err != nil {
		t.Fatalf("Search location: %v", err)
	}
	if len(hits) != 1 || hits[0].Match != "location" {
		t.Fatalf("unexpected location hits: %+v", hits)
	}

	detail, err := client.ArtistDetail(1)
	if err != nil {
		t.Fatalf("ArtistDetail: %v", err)
	}
	if len(detail.Concerts) != 1 || detail.Concerts[0].City != "London" {
		t.Fatalf("unexpected concerts: %+v", detail.Concerts)
	}
	if detail.Concerts[0].Dates[0] != "10-02-2020" {
		t.Fatalf("date star not trimmed: %+v", detail.Concerts[0].Dates)
	}
}

func TestArtistByIDNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/artists":
			_ = json.NewEncoder(w).Encode([]models.Artist{})
		case "/relation":
			_ = json.NewEncoder(w).Encode(models.RelationsResponse{Index: nil})
		case "/locations":
			_ = json.NewEncoder(w).Encode(models.LocationsResponse{Index: nil})
		case "/dates":
			_ = json.NewEncoder(w).Encode(models.DatesResponse{Index: nil})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.ArtistByID(99)
	if err == nil {
		t.Fatal("expected not found error")
	}
}
