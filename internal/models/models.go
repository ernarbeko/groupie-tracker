package models

// Artist represents a band or artist from the API.
type Artist struct {
	ID           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Locations    string   `json:"locations"`
	ConcertDates string   `json:"concertDates"`
	Relations    string   `json:"relations"`
}

// LocationEntry holds concert locations for one artist.
type LocationEntry struct {
	ID        int      `json:"id"`
	Locations []string `json:"locations"`
	Dates     string   `json:"dates"`
}

// LocationsResponse wraps the locations index endpoint.
type LocationsResponse struct {
	Index []LocationEntry `json:"index"`
}

// DateEntry holds concert dates for one artist.
type DateEntry struct {
	ID    int      `json:"id"`
	Dates []string `json:"dates"`
}

// DatesResponse wraps the dates index endpoint.
type DatesResponse struct {
	Index []DateEntry `json:"index"`
}

// RelationEntry links locations to concert dates for one artist.
type RelationEntry struct {
	ID             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// RelationsResponse wraps the relation index endpoint.
type RelationsResponse struct {
	Index []RelationEntry `json:"index"`
}

// ConcertStop is a cleaned location with its dates for display.
type ConcertStop struct {
	Location string
	City     string
	Country  string
	Dates    []string
}

// ArtistDetail is an artist enriched with relation data.
type ArtistDetail struct {
	Artist
	Concerts []ConcertStop
	AllLocations   []string // из /locations, сырые данные
	AllDates       []string // из /dates, сырые данные
}

// SearchHit is a lightweight result returned by the search API.
type SearchHit struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Image        string   `json:"image"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Members      []string `json:"members"`
	Match        string   `json:"match"`
}
