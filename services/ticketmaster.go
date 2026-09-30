package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"Beego-event-explorer/models"
)

var ErrEventNotFound = errors.New("event not found")

// seam for mocking in tests.
type EventProvider interface {
	ListEvents(ctx context.Context, city, country, category string) ([]models.Event, error)
	GetEvent(ctx context.Context, id string) (models.Event, error)
}

type TicketmasterClient struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewTicketmasterClient(apiKey string) *TicketmasterClient {
	return &TicketmasterClient{
		APIKey:  apiKey,
		BaseURL: "https://app.ticketmaster.com/discovery/v2",
		HTTP:    &http.Client{Timeout: 8 * time.Second},
	}
}

// Ticketmaster JSON

type tmImage struct {
	URL   string `json:"url"`
	Ratio string `json:"ratio"`
	Width int    `json:"width"`
}

type tmEvent struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	URL    string    `json:"url"`
	Info   string    `json:"info"`
	Note   string    `json:"pleaseNote"`
	Images []tmImage `json:"images"`
	Dates  struct {
		Timezone string `json:"timezone"`
		Start    struct {
			LocalDate string `json:"localDate"`
			LocalTime string `json:"localTime"`
		} `json:"start"`
	} `json:"dates"`
	Classifications []struct {
		Segment struct {
			Name string `json:"name"`
		} `json:"segment"`
	} `json:"classifications"`
	Embedded struct {
		Venues []struct {
			Name string `json:"name"`
			City struct {
				Name string `json:"name"`
			} `json:"city"`
			Country struct {
				CountryCode string `json:"countryCode"`
			} `json:"country"`
		} `json:"venues"`
	} `json:"_embedded"`
}

type tmListResponse struct {
	Embedded struct {
		Events []tmEvent `json:"events"`
	} `json:"_embedded"`
}

// smallest 16:9 image
func pickImage(imgs []tmImage, minWidth int) string {
	best, bestW := "", 0
	for _, im := range imgs {
		if im.Ratio == "16_9" && im.Width >= minWidth && (best == "" || im.Width < bestW) {
			best, bestW = im.URL, im.Width
		}
	}
	if best != "" {
		return best
	}
	maxW := 0
	for _, im := range imgs {
		if im.Width > maxW {
			best, maxW = im.URL, im.Width
		}
	}
	return best
}

func formatDate(localDate string) string {
	if localDate == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", localDate)
	if err != nil {
		return localDate
	}
	return t.Format("Mon, 02 Jan 2006")
}

func formatTime(localTime string) string {
	if localTime == "" {
		return ""
	}
	t, err := time.Parse("15:04:05", localTime)
	if err != nil {
		return ""
	}
	return t.Format("3:04 PM")
}

func (e tmEvent) toModel() models.Event {
	ev := models.Event{
		ID:        e.ID,
		Name:      e.Name,
		TicketURL: e.URL,
		ImageURL:  pickImage(e.Images, 500),
		HeroURL:   pickImage(e.Images, 1000),
		Date:      formatDate(e.Dates.Start.LocalDate),
		Time:      formatTime(e.Dates.Start.LocalTime),
		Timezone:  e.Dates.Timezone,
	}
	if len(e.Embedded.Venues) > 0 {
		v := e.Embedded.Venues[0]
		ev.Venue = v.Name
		ev.City = v.City.Name
		ev.CountryCode = v.Country.CountryCode
	}
	if len(e.Classifications) > 0 {
		if name := e.Classifications[0].Segment.Name; name != "Undefined" {
			ev.Category = name
		}
	}
	ev.Description = e.Info
	if ev.Description == "" {
		ev.Description = e.Note
	}
	return ev
}

func (c *TicketmasterClient) get(ctx context.Context, path string, q url.Values, out any) (int, error) {
	q.Set("apikey", c.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return 0, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("ticketmaster returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, json.NewDecoder(resp.Body).Decode(out)
}

func (c *TicketmasterClient) ListEvents(ctx context.Context, city, country, category string) ([]models.Event, error) {
	q := url.Values{}
	q.Set("city", city)
	q.Set("countryCode", country)
	q.Set("classificationName", category)
	q.Set("size", "6")

	var data tmListResponse
	if _, err := c.get(ctx, "/events.json", q, &data); err != nil {
		return nil, err
	}

	events := make([]models.Event, 0, len(data.Embedded.Events))
	for _, e := range data.Embedded.Events {
		events = append(events, e.toModel())
	}
	return events, nil // empty slice is valid
}

func (c *TicketmasterClient) GetEvent(ctx context.Context, id string) (models.Event, error) {
	var data tmEvent
	status, err := c.get(ctx, "/events/"+url.PathEscape(id)+".json", url.Values{}, &data)
	if err != nil {
		if status == http.StatusNotFound {
			return models.Event{}, ErrEventNotFound
		}
		return models.Event{}, err
	}
	return data.toModel(), nil
}
