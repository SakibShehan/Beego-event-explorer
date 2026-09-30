package models

type Event struct {
	ID          string
	Name        string
	ImageURL    string
	HeroURL     string
	Date        string
	Time        string
	Timezone    string
	Venue       string
	City        string
	CountryCode string
	Category    string
	Description string
	TicketURL   string
}

// One section can fail while the other still succeeds.
type ListingResult struct {
	Music     []Event
	Sports    []Event
	MusicErr  string
	SportsErr string
}
