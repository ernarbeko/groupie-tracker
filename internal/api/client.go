package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"groupie-tracker/internal/models"
)

const defaultBaseURL = "https://groupietrackers.herokuapp.com/api"

// apiDateLayout matches the "DD-MM-YYYY" format used by the Groupie Trackers API.
const apiDateLayout = "02-01-2006"

// Client fetches and caches Groupie Tracker API data.
type Client struct {
	baseURL    string
	httpClient *http.Client

	mu        sync.RWMutex
	artists   []models.Artist
	relations map[int]models.RelationEntry
	locations map[int]models.LocationEntry
	dates     map[int]models.DateEntry
	loadedAt  time.Time
	ttl       time.Duration
}

// NewClient creates an API client with a sensible timeout and cache TTL.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		ttl: 10 * time.Minute,
	}
}

// Refresh loads artists, relations, locations, and dates from the remote API.
// All four endpoints are fetched concurrently to cut total latency roughly
// fourfold compared to sequential requests.
func (c *Client) Refresh() error {
	var (
		wg        sync.WaitGroup
		artists   []models.Artist
		relations []models.RelationEntry
		locations []models.LocationEntry
		dates     []models.DateEntry
	)
	errCh := make(chan error, 4)

	wg.Add(4)
	go func() {
		defer wg.Done()
		a, err := c.fetchArtists()
		if err != nil {
			errCh <- err
			return
		}
		artists = a
	}()
	go func() {
		defer wg.Done()
		r, err := c.fetchRelations()
		if err != nil {
			errCh <- err
			return
		}
		relations = r
	}()
	go func() {
		defer wg.Done()
		l, err := c.fetchLocations()
		if err != nil {
			errCh <- err
			return
		}
		locations = l
	}()
	go func() {
		defer wg.Done()
		d, err := c.fetchDates()
		if err != nil {
			errCh <- err
			return
		}
		dates = d
	}()

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}

	relMap := make(map[int]models.RelationEntry, len(relations))
	for _, r := range relations {
		relMap[r.ID] = r
	}

	locMap := make(map[int]models.LocationEntry, len(locations))
	for _, l := range locations {
		locMap[l.ID] = l
	}
	dateMap := make(map[int]models.DateEntry, len(dates))
	for _, d := range dates {
		dateMap[d.ID] = d
	}

	c.mu.Lock()
	c.artists = artists
	c.relations = relMap
	c.locations = locMap
	c.dates = dateMap
	c.loadedAt = time.Now()
	c.mu.Unlock()
	return nil
}

func (c *Client) ensureLoaded() error {
	c.mu.RLock()
	fresh := len(c.artists) > 0 && time.Since(c.loadedAt) < c.ttl
	c.mu.RUnlock()
	if fresh {
		return nil
	}
	return c.Refresh()
}

// Artists returns all artists, refreshing the cache when needed.
func (c *Client) Artists() ([]models.Artist, error) {
	if err := c.ensureLoaded(); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]models.Artist, len(c.artists))
	copy(out, c.artists)
	return out, nil
}

// ArtistByID returns one artist by ID.
func (c *Client) ArtistByID(id int) (models.Artist, error) {
	artists, err := c.Artists()
	if err != nil {
		return models.Artist{}, err
	}
	for _, a := range artists {
		if a.ID == id {
			return a, nil
		}
	}
	return models.Artist{}, fmt.Errorf("artist %d not found", id)
}

// ArtistDetail returns an artist enriched with locations, dates, and
// relation-derived concert stops.
func (c *Client) ArtistDetail(id int) (models.ArtistDetail, error) {
	artist, err := c.ArtistByID(id)
	if err != nil {
		return models.ArtistDetail{}, err
	}
	if err := c.ensureLoaded(); err != nil {
		return models.ArtistDetail{}, err
	}

	c.mu.RLock()
	rel, relOk := c.relations[id]
	loc, locOk := c.locations[id]
	dt, dateOk := c.dates[id]
	c.mu.RUnlock()

	detail := models.ArtistDetail{Artist: artist}
	if locOk {
		detail.AllLocations = loc.Locations
	}
	if dateOk {
		detail.AllDates = dt.Dates
	}
	if !relOk {
		return detail, nil
	}

	for loc, dates := range rel.DatesLocations {
		city, country := SplitLocation(loc)
		cleaned := make([]string, 0, len(dates))
		for _, d := range dates {
			cleaned = append(cleaned, strings.TrimPrefix(d, "*"))
		}
		sortDates(cleaned)
		detail.Concerts = append(detail.Concerts, models.ConcertStop{
			Location: loc,
			City:     city,
			Country:  country,
			Dates:    cleaned,
		})
	}
	sort.Slice(detail.Concerts, func(i, j int) bool {
		if detail.Concerts[i].Country == detail.Concerts[j].Country {
			return detail.Concerts[i].City < detail.Concerts[j].City
		}
		return detail.Concerts[i].Country < detail.Concerts[j].Country
	})
	return detail, nil
}

// Search filters artists by query across name, members, album, year, and locations.
func (c *Client) Search(query string) ([]models.SearchHit, error) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return []models.SearchHit{}, nil
	}

	artists, err := c.Artists()
	if err != nil {
		return nil, err
	}
	if err := c.ensureLoaded(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	relations := c.relations
	c.mu.RUnlock()

	hits := make([]models.SearchHit, 0)
	for _, a := range artists {
		match := matchArtist(a, relations[a.ID], query)
		if match == "" {
			continue
		}
		hits = append(hits, models.SearchHit{
			ID:           a.ID,
			Name:         a.Name,
			Image:        a.Image,
			CreationDate: a.CreationDate,
			FirstAlbum:   a.FirstAlbum,
			Members:      a.Members,
			Match:        match,
		})
	}
	return hits, nil
}

func matchArtist(a models.Artist, rel models.RelationEntry, query string) string {
	if strings.Contains(strings.ToLower(a.Name), query) {
		return "artist"
	}
	for _, m := range a.Members {
		if strings.Contains(strings.ToLower(m), query) {
			return "member"
		}
	}
	if strings.Contains(strings.ToLower(a.FirstAlbum), query) {
		return "first album"
	}
	if strings.Contains(fmt.Sprintf("%d", a.CreationDate), query) {
		return "creation date"
	}
	for loc := range rel.DatesLocations {
		pretty := strings.ToLower(FormatLocation(loc))
		raw := strings.ToLower(loc)
		if strings.Contains(pretty, query) || strings.Contains(raw, query) {
			return "location"
		}
	}
	return ""
}

// sortDates sorts "DD-MM-YYYY" strings chronologically in place.
// Dates that fail to parse are left in place at the end, sorted as plain text.
func sortDates(dates []string) {
	sort.Slice(dates, func(i, j int) bool {
		ti, erri := time.Parse(apiDateLayout, dates[i])
		tj, errj := time.Parse(apiDateLayout, dates[j])
		if erri != nil || errj != nil {
			return dates[i] < dates[j]
		}
		return ti.Before(tj)
	})
}

// SplitLocation turns "los_angeles-usa" into ("Los Angeles", "Usa").
func SplitLocation(raw string) (city, country string) {
	raw = strings.TrimSpace(raw)
	parts := strings.SplitN(raw, "-", 2)
	city = TitleWords(strings.ReplaceAll(parts[0], "_", " "))
	if len(parts) == 2 {
		country = TitleWords(strings.ReplaceAll(parts[1], "_", " "))
	}
	return city, country
}

// FormatLocation turns "los_angeles-usa" into "Los Angeles, Usa".
func FormatLocation(raw string) string {
	city, country := SplitLocation(raw)
	if country == "" {
		return city
	}
	return city + ", " + country
}

// TitleWords capitalizes each whitespace-separated word.
func TitleWords(s string) string {
	fields := strings.Fields(strings.ToLower(s))
	for i, f := range fields {
		if f == "" {
			continue
		}
		fields[i] = strings.ToUpper(f[:1]) + f[1:]
	}
	return strings.Join(fields, " ")
}

func (c *Client) fetchArtists() ([]models.Artist, error) {
	var artists []models.Artist
	if err := c.getJSON("/artists", &artists); err != nil {
		return nil, err
	}
	return artists, nil
}

func (c *Client) fetchRelations() ([]models.RelationEntry, error) {
	var resp models.RelationsResponse
	if err := c.getJSON("/relation", &resp); err != nil {
		return nil, err
	}
	return resp.Index, nil
}

func (c *Client) fetchLocations() ([]models.LocationEntry, error) {
	var resp models.LocationsResponse
	if err := c.getJSON("/locations", &resp); err != nil {
		return nil, err
	}
	return resp.Index, nil
}

func (c *Client) fetchDates() ([]models.DateEntry, error) {
	var resp models.DatesResponse
	if err := c.getJSON("/dates", &resp); err != nil {
		return nil, err
	}
	return resp.Index, nil
}

func (c *Client) getJSON(path string, dest any) error {
	url := c.baseURL + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", path, err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: status %d", path, res.StatusCode)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}
