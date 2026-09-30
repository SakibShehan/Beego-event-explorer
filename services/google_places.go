package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"Beego-event-explorer/models"
)

var ErrPlaceNotFound = errors.New("place not found")

type LocationProvider interface {
	Autocomplete(ctx context.Context, input, sessionToken string) ([]models.Suggestion, error)
	PlaceCity(ctx context.Context, placeID, sessionToken string) (models.Location, error)
}

type GoogleClient struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func NewGoogleClient(apiKey string) *GoogleClient {
	return &GoogleClient{
		APIKey:  apiKey,
		BaseURL: "https://places.googleapis.com",
		HTTP:    &http.Client{Timeout: 6 * time.Second},
	}
}

// ---------- Autocomplete ----------

type acRequest struct {
	Input                string   `json:"input"`
	IncludedPrimaryTypes []string `json:"includedPrimaryTypes"`
	SessionToken         string   `json:"sessionToken"`
}

type acResponse struct {
	Suggestions []struct {
		PlacePrediction *struct {
			PlaceID string `json:"placeId"`
			Text    struct {
				Text string `json:"text"`
			} `json:"text"`
		} `json:"placePrediction"`
	} `json:"suggestions"`
}

func (g *GoogleClient) Autocomplete(ctx context.Context, input, sessionToken string) ([]models.Suggestion, error) {
	body, _ := json.Marshal(acRequest{
		Input:                input,
		IncludedPrimaryTypes: []string{"(cities)"},
		SessionToken:         sessionToken,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.BaseURL+"/v1/places:autocomplete", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", g.APIKey)

	resp, err := g.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google autocomplete returned status %d", resp.StatusCode)
	}

	var data acResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	out := make([]models.Suggestion, 0, 5)
	for _, s := range data.Suggestions {
		if s.PlacePrediction == nil {
			continue
		}
		out = append(out, models.Suggestion{
			PlaceID: s.PlacePrediction.PlaceID,
			Text:    s.PlacePrediction.Text.Text,
		})
		if len(out) == 5 {
			break
		}
	}
	return out, nil
}

// ---------- Place details ----------

type placeResponse struct {
	AddressComponents []struct {
		LongText  string   `json:"longText"`
		ShortText string   `json:"shortText"`
		Types     []string `json:"types"`
	} `json:"addressComponents"`
}

func (g *GoogleClient) PlaceCity(ctx context.Context, placeID, sessionToken string) (models.Location, error) {
	u := g.BaseURL + "/v1/places/" + url.PathEscape(placeID) +
		"?sessionToken=" + url.QueryEscape(sessionToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return models.Location{}, err
	}
	req.Header.Set("X-Goog-Api-Key", g.APIKey)
	req.Header.Set("X-Goog-FieldMask", "addressComponents")

	resp, err := g.HTTP.Do(req)
	if err != nil {
		return models.Location{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
		return models.Location{}, ErrPlaceNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return models.Location{}, fmt.Errorf("google place details returned status %d", resp.StatusCode)
	}

	var data placeResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return models.Location{}, err
	}

	var city, countryCode, countryName string
	fallbackCity := ""
	for _, c := range data.AddressComponents {
		for _, t := range c.Types {
			switch t {
			case "locality":
				city = c.LongText
			case "postal_town", "administrative_area_level_2":
				if fallbackCity == "" {
					fallbackCity = c.LongText
				}
			case "country":
				countryCode = c.ShortText
				countryName = c.LongText
			}
		}
	}
	if city == "" {
		city = fallbackCity
	}
	if city == "" || countryCode == "" {
		return models.Location{}, ErrPlaceNotFound
	}

	return models.Location{
		PlaceID:     placeID,
		City:        city,
		CountryCode: countryCode,
		Label:       city + ", " + countryName,
	}, nil
}
