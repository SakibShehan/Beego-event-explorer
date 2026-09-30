package models

type Suggestion struct {
	PlaceID string `json:"placeId"`
	Text    string `json:"text"`
}

type Location struct {
	PlaceID     string `json:"placeId"`
	City        string `json:"city"`
	CountryCode string `json:"countryCode"`
	Label       string `json:"label"`
}
