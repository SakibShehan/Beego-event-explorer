package services

import (
	"context"
	"log"
	"time"

	"Beego-event-explorer/models"
)

type EventService struct {
	Provider EventProvider
}

func NewEventService(p EventProvider) *EventService {
	return &EventService{Provider: p}
}

// it will send through  channel
type categoryResult struct {
	Events []models.Event
	Err    error
}

// fetchAsync starts a goroutine 
func (s *EventService) fetchAsync(ctx context.Context, city, country, category string) <-chan categoryResult {
	ch := make(chan categoryResult, 1)

	go func() {
		start := time.Now()
		log.Printf("[events] %s: started for %s, %s", category, city, country)

		events, err := s.Provider.ListEvents(ctx, city, country, category)

		if err != nil {
			log.Printf("[events] %s: failed after %s: %v", category, time.Since(start), err)
		} else {
			log.Printf("[events] %s: %d events in %s", category, len(events), time.Since(start))
		}
		ch <- categoryResult{Events: events, Err: err}
	}()

	return ch
}

// concurrency
func (s *EventService) Listing(ctx context.Context, city, country string) models.ListingResult {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	//Start both goroutines first
	musicCh := s.fetchAsync(ctx, city, country, "Music")
	sportsCh := s.fetchAsync(ctx, city, country, "Sports")

	// wait for both results
	musicRes := <-musicCh
	sportsRes := <-sportsCh

	// successful section even if one failed.
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
