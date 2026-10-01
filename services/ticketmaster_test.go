package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"Beego-event-explorer/models"
)

// Compile-time check: the real client satisfies the interface the service depends on.
var _ EventProvider = (*TicketmasterClient)(nil)

const tmTestKey = "test-key"

// Mock Ticketmaster responses


const tmFullEventJSON = `{
  "id": "Z7r9jZ1A7abc",
  "name": "Strings Under the Stars",
  "url": "https://www.ticketmaster.ca/event/10006123ABCD",
  "info": "Bring a blanket.",
  "pleaseNote": "Doors at 6.",
  "images": [
    {"url": "https://img.example/small.jpg",  "ratio": "16_9", "width": 305},
    {"url": "https://img.example/medium.jpg", "ratio": "16_9", "width": 640},
    {"url": "https://img.example/large.jpg",  "ratio": "16_9", "width": 1024},
    {"url": "https://img.example/square.jpg", "ratio": "1_1",  "width": 2048}
  ],
  "dates": {"timezone": "America/Toronto", "start": {"localDate": "2027-02-13", "localTime": "19:30:00"}},
  "classifications": [{"segment": {"name": "Music"}}],
  "_embedded": {"venues": [
    {"name": "Sample Riverside Hall", "city": {"name": "Toronto"}, "country": {"countryCode": "CA"}}
  ]}
}`

const tmMinimalEventJSON = `{"id": "minimal-1", "name": "Mystery Show"}`

const tmListJSON = `{"_embedded": {"events": [` + tmFullEventJSON + `,` + tmMinimalEventJSON +
	`]}, "page": {"size": 6, "totalElements": 2}}`

// What the full fixture must turn into.
var tmFullEvent = models.Event{
	ID:          "Z7r9jZ1A7abc",
	Name:        "Strings Under the Stars",
	ImageURL:    "https://img.example/medium.jpg",
	HeroURL:     "https://img.example/large.jpg",
	Date:        "Sat, 13 Feb 2027",
	Time:        "7:30 PM",
	Timezone:    "America/Toronto",
	Venue:       "Sample Riverside Hall",
	City:        "Toronto",
	CountryCode: "CA",
	Category:    "Music",
	Description: "Bring a blanket.",
	TicketURL:   "https://www.ticketmaster.ca/event/10006123ABCD",
}

var tmMinimalEvent = models.Event{ID: "minimal-1", Name: "Mystery Show"}


// Fake Ticketmaster server


// the fake server saw for one request.
type tmCall struct {
	method  string
	path    string // decoded path
	escaped string 
	query   url.Values
}

// starts a fake Ticketmaster
func newTMServer(t *testing.T, status int, body string) (*httptest.Server, <-chan tmCall) {
	t.Helper()
	calls := make(chan tmCall, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls <- tmCall{r.Method, r.URL.Path, r.URL.EscapedPath(), r.URL.Query()}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, calls
}

func newTMClient(baseURL string) *TicketmasterClient {
	return &TicketmasterClient{
		APIKey:  tmTestKey,
		BaseURL: baseURL,
		HTTP:    &http.Client{Timeout: 5 * time.Second},
	}
}

func mustTakeCall(t *testing.T, calls <-chan tmCall) tmCall {
	t.Helper()
	select {
	case c := <-calls:
		return c
	default:
		t.Fatal("the fake server received no request")
		return tmCall{}
	}
}


// Constructor


func TestNewTicketmasterClient(t *testing.T) {
	c := NewTicketmasterClient("abc")

	if c.APIKey != "abc" {
		t.Errorf("APIKey = %q, want %q", c.APIKey, "abc")
	}
	if c.BaseURL != "https://app.ticketmaster.com/discovery/v2" {
		t.Errorf("BaseURL = %q", c.BaseURL)
	}
	if c.HTTP == nil || c.HTTP.Timeout != 8*time.Second {
		t.Errorf("expected an HTTP client with an 8 second timeout, got %+v", c.HTTP)
	}
}


// ListEvents


func TestTicketmasterClient_ListEvents_SendsCorrectRequest(t *testing.T) {
	tests := []struct {
		name                    string
		city, country, category string
	}{
		{"simple city", "Toronto", "CA", "Music"},
		{"city with a space", "New York", "US", "Sports"},
		{"non-ASCII city", "São Paulo", "BR", "Music"},
		{"special characters are encoded, not interpreted", "A&B=C", "US", "Music"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, calls := newTMServer(t, http.StatusOK, tmListJSON)
			client := newTMClient(srv.URL)

			if _, err := client.ListEvents(context.Background(), tc.city, tc.country, tc.category); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			call := mustTakeCall(t, calls)
			if call.method != http.MethodGet {
				t.Errorf("method = %s, want GET", call.method)
			}
			if call.path != "/events.json" {
				t.Errorf("path = %q, want /events.json", call.path)
			}
			want := url.Values{
				"city":               {tc.city},
				"countryCode":        {tc.country},
				"classificationName": {tc.category},
				"size":               {"6"},
				"apikey":             {tmTestKey},
			}
			if !reflect.DeepEqual(call.query, want) {
				t.Fatalf("query = %v, want %v", call.query, want)
			}
		})
	}
}

func TestTicketmasterClient_ListEvents_ParsesEvents(t *testing.T) {
	srv, _ := newTMServer(t, http.StatusOK, tmListJSON)
	client := newTMClient(srv.URL)

	got, err := client.ListEvents(context.Background(), "Toronto", "CA", "Music")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []models.Event{tmFullEvent, tmMinimalEvent}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("events =\n  %+v\nwant\n  %+v", got, want)
	}
}

func TestTicketmasterClient_ListEvents_Responses(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		body      string
		wantCount int
		wantErr   string // substring of the error; "" means success
	}{
		{"two events", 200, tmListJSON, 2, ""},
		{"no _embedded key means no events", 200, `{"page":{"totalElements":0}}`, 0, ""},
		{"empty events array", 200, `{"_embedded":{"events":[]}}`, 0, ""},
		{"invalid api key", 401, `{"fault":{"faultstring":"Invalid ApiKey"}}`, 0, "status 401"},
		{"rate limited", 429, `{}`, 0, "status 429"},
		{"server error", 500, `oops`, 0, "status 500"},
		{"bad gateway with an empty body", 502, ``, 0, "status 502"},
		{"malformed json", 200, `{not json`, 0, "invalid character"},
		{"empty body", 200, ``, 0, "EOF"},
		{"wrong json type", 200, `{"_embedded":"oops"}`, 0, "cannot unmarshal"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newTMServer(t, tc.status, tc.body)
			client := newTMClient(srv.URL)

			got, err := client.ListEvents(context.Background(), "Toronto", "CA", "Music")

			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected an error containing %q, got none", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
				}
				if got != nil {
					t.Fatalf("events must be nil on error, got %v", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("an empty result must be an empty list, not nil")
			}
			if len(got) != tc.wantCount {
				t.Fatalf("got %d events, want %d", len(got), tc.wantCount)
			}
		})
	}
}


// Network-level failures (both ListEvents and GetEvent)


func TestTicketmasterClient_TransportFailures(t *testing.T) {
	type setup func(t *testing.T) (*TicketmasterClient, context.Context)

	cases := []struct {
		name   string
		setup  setup
		wantIs error // optional: the error must wrap this one
	}{
		{
			name: "server is down",
			setup: func(t *testing.T) (*TicketmasterClient, context.Context) {
				srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
				base := srv.URL
				srv.Close() // nobody is listening any more
				return newTMClient(base), context.Background()
			},
		},
		{
			name: "request times out",
			setup: func(t *testing.T) (*TicketmasterClient, context.Context) {
				slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					select {
					case <-r.Context().Done():
					case <-time.After(3 * time.Second):
					}
				}))
				t.Cleanup(slow.Close)
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				t.Cleanup(cancel)
				return newTMClient(slow.URL), ctx
			},
			wantIs: context.DeadlineExceeded,
		},
		{
			name: "context already cancelled",
			setup: func(t *testing.T) (*TicketmasterClient, context.Context) {
				srv, _ := newTMServer(t, http.StatusOK, tmListJSON)
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return newTMClient(srv.URL), ctx
			},
			wantIs: context.Canceled,
		},
		{
			name: "invalid base url",
			setup: func(t *testing.T) (*TicketmasterClient, context.Context) {
				return newTMClient("http://exa mple.com"), context.Background()
			},
		},
	}

	operations := []struct {
		name string
		run  func(ctx context.Context, c *TicketmasterClient) error
	}{
		{"ListEvents", func(ctx context.Context, c *TicketmasterClient) error {
			_, err := c.ListEvents(ctx, "Toronto", "CA", "Music")
			return err
		}},
		{"GetEvent", func(ctx context.Context, c *TicketmasterClient) error {
			_, err := c.GetEvent(ctx, "abc123")
			return err
		}},
	}

	for _, tc := range cases {
		for _, op := range operations {
			t.Run(tc.name+"/"+op.name, func(t *testing.T) {
				client, ctx := tc.setup(t)
				err := op.run(ctx, client)

				if err == nil {
					t.Fatal("expected an error, got none")
				}
				if tc.wantIs != nil && !errors.Is(err, tc.wantIs) {
					t.Fatalf("error %q should wrap %v", err, tc.wantIs)
				}
				if errors.Is(err, ErrEventNotFound) {
					t.Fatal("a transport failure must not be reported as 'event not found'")
				}
				// Errors are logged by the service, so they must never contain the API key.
				if strings.Contains(err.Error(), tmTestKey) {
					t.Fatalf("error leaks the API key: %q", err)
				}
			})
		}
	}
}


// GetEvent


func TestTicketmasterClient_GetEvent_Success(t *testing.T) {
	srv, calls := newTMServer(t, http.StatusOK, tmFullEventJSON)
	client := newTMClient(srv.URL)

	got, err := client.GetEvent(context.Background(), "Z7r9jZ1A7abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	call := mustTakeCall(t, calls)
	if call.method != http.MethodGet {
		t.Errorf("method = %s, want GET", call.method)
	}
	if call.path != "/events/Z7r9jZ1A7abc.json" {
		t.Errorf("path = %q, want /events/Z7r9jZ1A7abc.json", call.path)
	}
	if want := (url.Values{"apikey": {tmTestKey}}); !reflect.DeepEqual(call.query, want) {
		t.Errorf("query = %v, want %v", call.query, want)
	}
	if !reflect.DeepEqual(got, tmFullEvent) {
		t.Fatalf("event =\n  %+v\nwant\n  %+v", got, tmFullEvent)
	}
}

func TestTicketmasterClient_GetEvent_Failures(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		body         string
		wantNotFound bool
		wantText     string
	}{
		{"unknown event id", 404, `{"errors":[{"code":"DIS1004","detail":"Event not found"}]}`, true, ""},
		{"invalid api key", 401, `{}`, false, "status 401"},
		{"rate limited", 429, `{}`, false, "status 429"},
		{"server error", 500, `oops`, false, "status 500"},
		{"malformed json", 200, `{broken`, false, "invalid character"},
		{"empty body", 200, ``, false, "EOF"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newTMServer(t, tc.status, tc.body)
			client := newTMClient(srv.URL)

			got, err := client.GetEvent(context.Background(), "abc123")

			if err == nil {
				t.Fatal("expected an error, got none")
			}
			if !reflect.DeepEqual(got, models.Event{}) {
				t.Fatalf("a failed lookup must return an empty event, got %+v", got)
			}
			if tc.wantNotFound {
				if !errors.Is(err, ErrEventNotFound) {
					t.Fatalf("expected ErrEventNotFound, got %v", err)
				}
				return
			}
			if errors.Is(err, ErrEventNotFound) {
				t.Fatalf("only a 404 may map to ErrEventNotFound, got it for: %v", err)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Fatalf("error = %q, want it to contain %q", err, tc.wantText)
			}
		})
	}
}

// A hostile event id must stay inside the path and never add query parameters.
func TestTicketmasterClient_GetEvent_EscapesTheID(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{"plain id", "abc123"},
		{"path traversal attempt", "../admin"},
		{"slash inside the id", "a/b"},
		{"query injection attempt", "a?apikey=stolen&x=1"},
		{"space", "a b"},
		{"percent sign", "50%"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv, calls := newTMServer(t, http.StatusOK, tmFullEventJSON)
			client := newTMClient(srv.URL)

			if _, err := client.GetEvent(context.Background(), tc.id); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			call := mustTakeCall(t, calls)
			if want := "/events/" + url.PathEscape(tc.id) + ".json"; call.escaped != want {
				t.Errorf("path on the wire = %q, want %q", call.escaped, want)
			}
			if n := strings.Count(call.escaped, "/"); n != 2 {
				t.Errorf("the id must stay one path segment, path has %d slashes: %q", n, call.escaped)
			}
			if want := (url.Values{"apikey": {tmTestKey}}); !reflect.DeepEqual(call.query, want) {
				t.Errorf("query = %v, want only the api key", call.query)
			}
		})
	}
}


// tmEvent.toModel: how Ticketmaster JSON becomes our Event


func TestTicketmasterEventToModel(t *testing.T) {
	tests := []struct {
		name string
		body string
		want models.Event
	}{
		{"full event", tmFullEventJSON, tmFullEvent},
		{"only id and name", tmMinimalEventJSON, tmMinimalEvent},
		{
			"no venue",
			`{"id":"v1","name":"No Venue","url":"https://www.ticketmaster.com/e",
			  "dates":{"timezone":"UTC","start":{"localDate":"2027-03-05","localTime":"09:05:00"}},
			  "classifications":[{"segment":{"name":"Sports"}}]}`,
			models.Event{
				ID: "v1", Name: "No Venue", TicketURL: "https://www.ticketmaster.com/e",
				Date: "Fri, 05 Mar 2027", Time: "9:05 AM", Timezone: "UTC", Category: "Sports",
			},
		},
		{
			"the first venue wins",
			`{"id":"v2","name":"Two Venues","_embedded":{"venues":[
			  {"name":"First Hall","city":{"name":"Toronto"},"country":{"countryCode":"CA"}},
			  {"name":"Second Hall","city":{"name":"Ottawa"},"country":{"countryCode":"CA"}}]}}`,
			models.Event{ID: "v2", Name: "Two Venues", Venue: "First Hall", City: "Toronto", CountryCode: "CA"},
		},
		{
			"Undefined segment gives no category",
			`{"id":"c1","name":"Odd","classifications":[{"segment":{"name":"Undefined"}}]}`,
			models.Event{ID: "c1", Name: "Odd"},
		},
		{
			"empty classifications give no category",
			`{"id":"c2","name":"Odd","classifications":[]}`,
			models.Event{ID: "c2", Name: "Odd"},
		},
		{
			"description falls back to pleaseNote",
			`{"id":"d1","name":"Note Only","pleaseNote":"Doors at 6."}`,
			models.Event{ID: "d1", Name: "Note Only", Description: "Doors at 6."},
		},
		{
			"info wins over pleaseNote",
			`{"id":"d2","name":"Both","info":"Main info","pleaseNote":"Doors at 6."}`,
			models.Event{ID: "d2", Name: "Both", Description: "Main info"},
		},
		{
			"malformed date keeps the raw text and drops the time",
			`{"id":"t1","name":"Soon","dates":{"start":{"localDate":"TBD","localTime":"later"}}}`,
			models.Event{ID: "t1", Name: "Soon", Date: "TBD"},
		},
		{
			"one medium image is used for the card and the hero",
			`{"id":"i1","name":"One Image","images":[{"url":"https://img.example/only.jpg","ratio":"16_9","width":640}]}`,
			models.Event{
				ID: "i1", Name: "One Image",
				ImageURL: "https://img.example/only.jpg", HeroURL: "https://img.example/only.jpg",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var raw tmEvent
			if err := json.Unmarshal([]byte(tc.body), &raw); err != nil {
				t.Fatalf("test fixture is not valid JSON: %v", err)
			}
			if got := raw.toModel(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("toModel() =\n  %+v\nwant\n  %+v", got, tc.want)
			}
		})
	}
}


// Small helpers


func tmImg(u, ratio string, width int) tmImage { return tmImage{URL: u, Ratio: ratio, Width: width} }

func TestTicketmasterPickImage(t *testing.T) {
	mixed := []tmImage{
		tmImg("a", "16_9", 300),
		tmImg("b", "16_9", 640),
		tmImg("c", "16_9", 1024),
		tmImg("d", "3_2", 2048),
	}
	tests := []struct {
		name     string
		imgs     []tmImage
		minWidth int
		want     string
	}{
		{"nil list", nil, 500, ""},
		{"empty list", []tmImage{}, 500, ""},
		{"smallest 16:9 image that is wide enough", mixed, 500, "b"},
		{"a larger minimum picks a larger image", mixed, 1000, "c"},
		{"exactly the minimum width is accepted",
			[]tmImage{tmImg("min", "16_9", 500), tmImg("big", "16_9", 800)}, 500, "min"},
		{"nothing wide enough falls back to the widest of any ratio", mixed, 1500, "d"},
		{"no 16:9 images falls back to the widest",
			[]tmImage{tmImg("x", "1_1", 100), tmImg("y", "3_2", 900)}, 500, "y"},
		{"only a small 16:9 image is still used",
			[]tmImage{tmImg("only", "16_9", 200)}, 500, "only"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickImage(tc.imgs, tc.minWidth); got != tc.want {
				t.Fatalf("pickImage(min %d) = %q, want %q", tc.minWidth, got, tc.want)
			}
		})
	}
}

func TestTicketmasterFormatDate(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"normal date", "2027-02-13", "Sat, 13 Feb 2027"},
		{"day is zero-padded", "2027-03-05", "Fri, 05 Mar 2027"},
		{"leap day", "2028-02-29", "Tue, 29 Feb 2028"},
		{"empty input", "", ""},
		{"malformed input is returned unchanged", "TBD", "TBD"},
		{"wrong format is returned unchanged", "13/02/2027", "13/02/2027"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatDate(tc.in); got != tc.want {
				t.Fatalf("formatDate(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTicketmasterFormatTime(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"evening", "19:30:00", "7:30 PM"},
		{"morning has no leading zero", "09:05:00", "9:05 AM"},
		{"midnight", "00:00:00", "12:00 AM"},
		{"noon", "12:00:00", "12:00 PM"},
		{"empty input", "", ""},
		{"malformed input", "7pm", ""},
		{"missing seconds", "19:30", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatTime(tc.in); got != tc.want {
				t.Fatalf("formatTime(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}