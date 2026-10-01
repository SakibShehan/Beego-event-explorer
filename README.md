# Event Explorer

Event Explorer is a Go web application for discovering upcoming **music** and **sports** events in any city.
You pick a city with Google-powered autocomplete, browse events loaded from Ticketmaster, open an event's details,
and continue to that event's ticket page on Ticketmaster through a validated, server-side redirect.

Built with Go, Beego v2, server-side rendered templates, goroutines and channels, a shared in-memory cache,
and table-driven unit tests with mocked API servers.


## Features

- City search with Google Places autocomplete
- Music and sports event discovery (up to 6 events each)
- Concurrent event fetching using goroutines and channels
- Mutex-protected in-memory cache with manual invalidation
- Safe ticket redirect to the event's Ticketmaster page
- Table-driven unit tests with mocked API servers

---

## Tech Stack

| Technology | Purpose |
|---|---|
| Go | Backend language |
| Beego v2 | Web framework: routing, controllers, template rendering |
| Google Places API (New) | City autocomplete and selected-city lookup |
| Ticketmaster Discovery API v2 | Event lists, event details and ticket links |
| `html/template` (Beego views) | Server-side rendered pages with shared header and footer partials |
| Goroutines and channels | Concurrent Music and Sports requests |
| `sync.Mutex` + map | Shared in-memory cache |
| HTML, CSS, vanilla JavaScript | Frontend. JavaScript is used only for the autocomplete box |
| `net/http/httptest` | Mock servers for unit tests |

---


## Project Structure

```
Beego-event-explorer/
├── conf/
│   └── app.conf                   
├── controllers/
│   ├── home.go                    
│   ├── events.go                   
│   ├── redirect.go                 
│   ├── api.go                     
│   └── cache.go                   
├── models/
│   ├── event.go                    
│   └── location.go                
├── routers/
│   └── router.go                   
├── services/
│   ├── cache.go                    
│   ├── event_service.go            
│   ├── google_places.go           
│   ├── ticketmaster.go            
│   ├── ticket.go                  
│   └── *_test.go                  #Test files for each file
├── views/
│   ├── partials/
│   │   ├── header.tpl              
│   │   ├── footer.tpl              
│   │   └── event_card.tpl          
│   ├── home.tpl
│   ├── listing.tpl
│   ├── details.tpl
│   ├── demo_ticket.tpl             
│   └── error.tpl
├── static/
│   ├── css/style.css
│   ├── js/home.js                  
│   └── img/
├── main.go                         
├── .env.example
└── go.mod
```


---

## Setup

### Requirements

- Go 1.21 or newer
- A **Ticketmaster** API key (free): <https://developer-account.ticketmaster.com/user/login>
- A **Google Places API (New)** key. The Google Cloud project needs billing enabled and **Places API (New)** turned on.
- Optional: [`bee`](https://github.com/beego/bee) for auto-reload during development

### Installation

```bash
git clone https://github.com/SakibShehan/Beego-event-explorer.git
cd Beego-event-explorer

go mod tidy
```

### Environment file

Create a `.env` file in the project root. It is git-ignored, so your keys are never committed.

```env
GOOGLE_API_KEY=your_google_places_api_key
TICKETMASTER_API_KEY=your_ticketmaster_api_key
```

### Run

```bash
go run .
```

or, with auto-reload:

```bash
bee run
```

Open <http://localhost:8080>. On startup the terminal shows the active configuration, for example:

```
[config] cache: no automatic expiry
[config] ticket mode: live
```

---

## Configuration

All settings come from `.env`. Only the two keys are required.

| Variable | Required | Default | Description |
|---|---|---|---|
| `GOOGLE_API_KEY` | yes | none | Google Places (New) key for city autocomplete and lookup |
| `TICKETMASTER_API_KEY` | yes | none | Ticketmaster Discovery API key |
| `TICKET_MODE` | no | `live` | `live` sends **View tickets** to the real Ticketmaster page. `mock` sends it to the local demo page `/demo/tickets/:eventId` |
| `TICKET_ALLOWED_HOSTS` | no | empty | Extra approved ticket hostnames, comma separated, added to the built-in list |

Example with every option:

```env
GOOGLE_API_KEY=your_google_places_api_key
TICKETMASTER_API_KEY=your_ticketmaster_api_key
TICKET_MODE=live
TICKET_ALLOWED_HOSTS=

```

---

## Routes

Base URL: `http://localhost:8080`

### Pages (server-side rendered HTML)

| Method | Route | Description |
|---|---|---|
| GET | `/` | Home page with city search |
| GET | `/events?city=Toronto&countryCode=CA` | Listing with Music and Sports sections |
| GET | `/events/:eventId` | Event details. Direct links work |

### Ticket actions

| Method | Route | Description |
|---|---|---|
| GET | `/redirect/:eventId` | Backend action, not a page. Validates the event's ticket URL and answers with a **302** |
| GET | `/demo/tickets/:eventId` | Local ticket destination. **Mock mode only**; returns 404 in live mode |

### JSON API

| Method | Route | Query | Description |
|---|---|---|---|
| GET | `/api/locations/autocomplete` | `input` (3+ characters), `sessionToken` | Up to five city suggestions |
| GET | `/api/locations/:placeId` | `sessionToken` | City and country code for a selected place |

### Cache maintenance

| Method | Route | What it deletes |
|---|---|---|
| GET | `/delete` | The whole cache |
| GET | `/delete/music` or `/delete/sports` | That category, for every city |
| GET | `/delete/ca`, `/delete/us`, ... | Everything for a two-letter **country code** |
| GET | `/delete/toronto`, `/delete/new%20york`, ... | Everything for that **city** (Music and Sports) |


---

# Testing

Run all tests:

```bash
go test ./...
```

Verbose output:

```bash
go test ./services -v -count=1
```

Coverage:

```bash
go test ./services -cover -count=1
```

Testing includes:

- Table-driven unit tests
- Mock HTTP servers for Google and Ticketmaster
- Cache hit, expiry and delete tests
- Concurrency tests for the goroutines
- Ticket link validation tests

Service layer coverage: **98.6%**

---

# Architecture Overview

```
Browser
   |
Router
   |
Controllers
   |
Services (cache, goroutines)
   |
   +----------------+
   |                |
Google API     Ticketmaster API
   |
Models
   |
Views
```

## Author

Built by **Sakib Hossen Shehan** ([@SakibShehan](https://github.com/SakibShehan)).