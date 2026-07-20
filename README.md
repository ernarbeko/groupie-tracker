# Groupie Tracker

A Go web app that consumes the [Groupie Trackers API](https://groupietrackers.herokuapp.com/api) and displays artists, members, albums, and concert relations.

## Features

- Artist gallery with images and key facts
- Artist detail pages with members and concert location/date tables
- **Live search** (client → server → JSON response) by artist name, member, creation year, first album, or concert location
- Cached API data with graceful error pages (404 / 502 / method errors)
- Panic recovery so the server stays up

## Requirements

- Go 1.21+ (standard library only)

## Run

```bash
go run .
```

Open [http://localhost:8080](http://localhost:8080).

Optional environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `ADDR` | `:8080` | Listen address |
| `API_URL` | `https://groupietrackers.herokuapp.com/api` | API base URL |

## Test

```bash
go test ./...
```

## Project layout

```
main.go
internal/api/        # API client, cache, search
internal/handlers/   # HTTP handlers and templates
internal/models/     # Data structures
templates/           # HTML templates
static/              # CSS and JS
```

## Client-server event

Typing in the search box triggers a `GET /api/search?q=…` request. The server filters artists and returns JSON; the page updates without a full reload.
