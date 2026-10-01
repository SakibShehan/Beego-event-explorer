package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"Beego-event-explorer/models"
)

// Compile-time check: the real client satisfies the interface the controller uses.
var _ LocationProvider = (*GoogleClient)(nil)

const googleTestKey = "google-test-key"


// Fake Google server and small JSON builders


// server saw it for one request.
type gCall struct {
	method, path, rawQuery, body string
	header                       http.Header
}

// starts a fake Google 
func newGoogleServer(t *testing.T, status int, body string) (*GoogleClient, <-chan gCall) {
	t.Helper()
	calls := make(chan gCall, 5)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		calls <- gCall{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, string(b), r.Header.Clone()}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &GoogleClient{APIKey: googleTestKey, BaseURL: srv.URL, HTTP: srv.Client()}, calls
}

func sug(id, text string) models.Suggestion { return models.Suggestion{PlaceID: id, Text: text} }

func predictionJSON(id, text string) string {
	return fmt.Sprintf(`{"placePrediction":{"placeId":%q,"text":{"text":%q}}}`, id, text)
}

func comp(long, short string, types ...string) string {
	quoted := make([]string, len(types))
	for i, ty := range types {
		quoted[i] = `"` + ty + `"`
	}
	return fmt.Sprintf(`{"longText":%q,"shortText":%q,"types":[%s]}`, long, short, strings.Join(quoted, ","))
}

func addressJSON(components ...string) string {
	return `{"addressComponents":[` + strings.Join(components, ",") + `]}`
}


// Constructor
func TestNewGoogleClient(t *testing.T) {
	c := NewGoogleClient("abc")

	if c.APIKey != "abc" || c.BaseURL != "https://places.googleapis.com" {
		t.Errorf("unexpected client: key=%q base=%q", c.APIKey, c.BaseURL)
	}
	if c.HTTP == nil || c.HTTP.Timeout != 6*time.Second {
		t.Errorf("expected an HTTP client with a 6 second timeout, got %+v", c.HTTP)
	}
}

// Autocomplete


func TestGoogleClient_Autocomplete_Request(t *testing.T) {
	client, calls := newGoogleServer(t, http.StatusOK, `{}`)

	if _, err := client.Autocomplete(context.Background(), "toro", "session-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := <-calls

	if c.method != http.MethodPost || c.path != "/v1/places:autocomplete" {
		t.Errorf("request = %s %s, want POST /v1/places:autocomplete", c.method, c.path)
	}
	if got := c.header.Get("X-Goog-Api-Key"); got != googleTestKey {
		t.Errorf("X-Goog-Api-Key = %q, want the client's key", got)
	}
	if got := c.header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}

	var body map[string]any
	if err := json.Unmarshal([]byte(c.body), &body); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	want := map[string]any{
		"input":                "toro",
		"includedPrimaryTypes": []any{"(cities)"},
		"sessionToken":         "session-1",
	}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("request body = %v, want %v", body, want)
	}
}

func TestGoogleClient_Autocomplete_Responses(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    []models.Suggestion
		wantErr string // substring of the error; "" means success
	}{
		{"two suggestions", 200,
			`{"suggestions":[` + predictionJSON("id1", "Toronto, ON, Canada") + `,` + predictionJSON("id2", "Toro, Spain") + `]}`,
			[]models.Suggestion{sug("id1", "Toronto, ON, Canada"), sug("id2", "Toro, Spain")}, ""},
		{"entries without a place prediction are skipped", 200,
			`{"suggestions":[{"queryPrediction":{"text":{"text":"toronto"}}},` + predictionJSON("id1", "Toronto") + `]}`,
			[]models.Suggestion{sug("id1", "Toronto")}, ""},
		{"no suggestions gives an empty list", 200, `{}`, []models.Suggestion{}, ""},
		{"invalid api key", 401, `{}`, nil, "status 401"},
		{"api not enabled or billing off", 403, `{}`, nil, "status 403"},
		{"server error", 500, `oops`, nil, "status 500"},
		{"malformed json", 200, `{nope`, nil, "invalid character"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newGoogleServer(t, tc.status, tc.body)

			got, err := client.Autocomplete(context.Background(), "toro", "session-1")

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want it to contain %q", err, tc.wantErr)
				}
				if got != nil {
					t.Fatalf("suggestions must be nil on error, got %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("suggestions = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGoogleClient_Autocomplete_LimitsToFive(t *testing.T) {
	parts := make([]string, 8)
	for i := range parts {
		parts[i] = predictionJSON(fmt.Sprintf("id%d", i+1), fmt.Sprintf("City %d", i+1))
	}
	client, _ := newGoogleServer(t, http.StatusOK, `{"suggestions":[`+strings.Join(parts, ",")+`]}`)

	got, err := client.Autocomplete(context.Background(), "city", "session-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 5 || got[0].PlaceID != "id1" || got[4].PlaceID != "id5" {
		t.Fatalf("expected the first 5 suggestions in order, got %v", got)
	}
}


// PlaceCity


func TestGoogleClient_PlaceCity_Request(t *testing.T) {
	tests := []struct{ name, placeID, token string }{
		{"plain values", "ChIJabc", "session-1"},
		{"hostile values stay inside their own slots", "a/b?x=1", "tok&x=1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, calls := newGoogleServer(t, http.StatusOK,
				addressJSON(comp("Toronto", "Toronto", "locality"), comp("Canada", "CA", "country")))

			if _, err := client.PlaceCity(context.Background(), tc.placeID, tc.token); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			c := <-calls

			if c.method != http.MethodGet {
				t.Errorf("method = %s, want GET", c.method)
			}
			if n := strings.Count(c.path, "/"); n != 3 {
				t.Errorf("the place id must stay one path segment, got %q", c.path)
			}
			q, _ := url.ParseQuery(c.rawQuery)
			if len(q) != 1 || q.Get("sessionToken") != tc.token {
				t.Errorf("query = %v, want only sessionToken=%q", q, tc.token)
			}
			if got := c.header.Get("X-Goog-Api-Key"); got != googleTestKey {
				t.Errorf("X-Goog-Api-Key = %q, want the client's key", got)
			}
			if got := c.header.Get("X-Goog-FieldMask"); got != "addressComponents" {
				t.Errorf("X-Goog-FieldMask = %q, want addressComponents", got)
			}
		})
	}
}

func TestGoogleClient_PlaceCity_Responses(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		want     models.Location
		wantErr  error  // the error must wrap this one
		wantText string // or: a generic error containing this text
	}{
		{"locality and country", 200,
			addressJSON(comp("Toronto", "Toronto", "locality", "political"),
				comp("Ontario", "ON", "administrative_area_level_1", "political"),
				comp("Canada", "CA", "country", "political")),
			models.Location{PlaceID: "p1", City: "Toronto", CountryCode: "CA", Label: "Toronto, Canada"}, nil, ""},
		{"postal_town is used when there is no locality", 200,
			addressJSON(comp("London", "London", "postal_town"), comp("United Kingdom", "GB", "country")),
			models.Location{PlaceID: "p1", City: "London", CountryCode: "GB", Label: "London, United Kingdom"}, nil, ""},
		{"administrative_area_level_2 is the last resort", 200,
			addressJSON(comp("Dhaka", "Dhaka", "administrative_area_level_2"), comp("Bangladesh", "BD", "country")),
			models.Location{PlaceID: "p1", City: "Dhaka", CountryCode: "BD", Label: "Dhaka, Bangladesh"}, nil, ""},
		{"locality wins over the fallbacks", 200,
			addressJSON(comp("Fallback Town", "x", "postal_town"), comp("Real City", "x", "locality"), comp("Canada", "CA", "country")),
			models.Location{PlaceID: "p1", City: "Real City", CountryCode: "CA", Label: "Real City, Canada"}, nil, ""},
		{"the first fallback wins", 200,
			addressJSON(comp("First", "x", "postal_town"), comp("Second", "x", "administrative_area_level_2"), comp("Canada", "CA", "country")),
			models.Location{PlaceID: "p1", City: "First", CountryCode: "CA", Label: "First, Canada"}, nil, ""},

		{"no country", 200, addressJSON(comp("Toronto", "Toronto", "locality")), models.Location{}, ErrPlaceNotFound, ""},
		{"no city", 200, addressJSON(comp("Canada", "CA", "country")), models.Location{}, ErrPlaceNotFound, ""},
		{"no address components", 200, `{}`, models.Location{}, ErrPlaceNotFound, ""},
		{"google says bad request", 400, `{}`, models.Location{}, ErrPlaceNotFound, ""},
		{"google says not found", 404, `{}`, models.Location{}, ErrPlaceNotFound, ""},

		{"api not enabled or billing off", 403, `{}`, models.Location{}, nil, "status 403"},
		{"server error", 500, `oops`, models.Location{}, nil, "status 500"},
		{"malformed json", 200, `{nope`, models.Location{}, nil, "invalid character"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client, _ := newGoogleServer(t, tc.status, tc.body)

			got, err := client.PlaceCity(context.Background(), "p1", "session-1")

			switch {
			case tc.wantErr != nil:
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("error = %v, want %v", err, tc.wantErr)
				}
			case tc.wantText != "":
				if err == nil || !strings.Contains(err.Error(), tc.wantText) {
					t.Fatalf("error = %v, want it to contain %q", err, tc.wantText)
				}
				if errors.Is(err, ErrPlaceNotFound) {
					t.Fatal("a Google outage must not be reported as 'place not found'")
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("location = %+v, want %+v", got, tc.want)
			}
		})
	}
}


// Network failure


func TestGoogleClient_ServerDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := &GoogleClient{APIKey: googleTestKey, BaseURL: srv.URL, HTTP: srv.Client()}
	srv.Close() // nobody is listening any more

	_, acErr := client.Autocomplete(context.Background(), "toro", "session-1")
	_, plErr := client.PlaceCity(context.Background(), "p1", "session-1")

	for name, err := range map[string]error{"Autocomplete": acErr, "PlaceCity": plErr} {
		if err == nil {
			t.Errorf("%s: expected an error, got none", name)
			continue
		}
		if errors.Is(err, ErrPlaceNotFound) {
			t.Errorf("%s: an outage must not be reported as 'place not found'", name)
		}
		if strings.Contains(err.Error(), googleTestKey) {
			t.Errorf("%s: error leaks the API key: %q", name, err)
		}
	}
}