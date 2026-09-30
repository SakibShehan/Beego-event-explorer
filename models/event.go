package models

type Event struct {
	ID          string
	Name        string
	ImageURL    string
	Date        string // already formatted, e.g. "Sun, 07 Feb 2027"
	Venue       string
	City        string
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
