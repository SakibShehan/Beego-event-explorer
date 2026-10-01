package services

import (
	"context"
	"log"
	"time"

	"Beego-event-explorer/models"
)

type EventService struct {
	Provider    EventProvider
	TicketHosts []string
	Cache       *EventCache
}

func NewEventService(p EventProvider) *EventService {
	return &EventService{
		Provider:    p,
		TicketHosts: DefaultTicketHosts,
		Cache:       NewEventCache(0), // 0 = entries stay until cleared
	}
}

// goroutine sends this structure back through its channel.
type categoryResult struct {
	Events []models.Event
	Err    error
}

// starts a goroutine for one category and returns the channel

func (s *EventService) fetchAsync(ctx context.Context, city, country, category string) <-chan categoryResult {
	ch := make(chan categoryResult, 1)

	go func() {
		key := CacheKey(city, country, category)

		// Cache a valid entry
		if s.Cache != nil {
			if events, ok := s.Cache.Get(key); ok {
				ch <- categoryResult{Events: events}
				return
			}
		}

		// if Miss or expired: fetch from Ticketmaster.
		start := time.Now()
		log.Printf("[events] %s: started for %s, %s", category, city, country)

		events, err := s.Provider.ListEvents(ctx, city, country, category)
		if err != nil {
	
			log.Printf("[events] %s: failed after %s: %v", category, time.Since(start), err)
			ch <- categoryResult{Err: err}
			return
		}
		log.Printf("[events] %s: %d events in %s", category, len(events), time.Since(start))

		// successful lists are stored.
		if s.Cache != nil {
			s.Cache.Set(key, events)
		}
		ch <- categoryResult{Events: events}
	}()

	return ch
}

// fetches Music and Sports concurrently.
func (s *EventService) Listing(ctx context.Context, city, country string) models.ListingResult {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Start goroutines 
	musicCh := s.fetchAsync(ctx, city, country, "Music")
	sportsCh := s.fetchAsync(ctx, city, country, "Sports")

	// wait for both results.
	musicRes := <-musicCh
	sportsRes := <-sportsCh

	// 3) Keep a successful section 
	var res models.ListingResult
	if musicRes.Err != nil {
		res.MusicErr = "Music events are unavailable right now. Please try again in a moment."
	} else {
		res.Music = musicRes.Events
	}
	if sportsRes.Err != nil {
		res.SportsErr = "Sports events are unavailable right now. Please try again in a moment."
	} else {
		res.Sports = sportsRes.Events
	}
	return res
}

// fetches one event by ID.
func (s *EventService) Details(ctx context.Context, id string) (models.Event, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return s.Provider.GetEvent(ctx, id)
}

// checks a provider ticket URL against the approved hosts.
func (s *EventService) ValidTicketURL(raw string) (string, error) {
	return ValidateTicketURL(raw, s.TicketHosts)
}