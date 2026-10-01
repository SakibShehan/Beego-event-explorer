package services

import (
	"log"
	"strings"
	"sync"
	"time"

	"Beego-event-explorer/models"
)

type cacheEntry struct {
	events    []models.Event
	expiresAt time.Time // never expires
}

// mutex-protected map of event lists.

type EventCache struct {
	mu    sync.Mutex
	items map[string]cacheEntry
	ttl   time.Duration
	now   func() time.Time 
}

func NewEventCache(ttl time.Duration) *EventCache {
	return &EventCache{
		items: make(map[string]cacheEntry),
		ttl:   ttl,
		now:   time.Now,
	}
}

// map key from city, country and category.

func CacheKey(city, country, category string) string {
	return strings.ToLower(strings.TrimSpace(city)) + "|" +
		strings.ToUpper(strings.TrimSpace(country)) + "|" +
		strings.ToLower(strings.TrimSpace(category))
}

// returns the cached events if the key exists 

func (c *EventCache) Get(key string) ([]models.Event, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.items[key]
	if !ok {
		log.Printf("[cache] MISS    %s", key)
		return nil, false
	}

	if entry.expiresAt.IsZero() {
		log.Printf("[cache] HIT     %s (no expiry)", key)
		return copyEvents(entry.events), true
	}

	now := c.now()
	if !now.Before(entry.expiresAt) {
		delete(c.items, key)
		log.Printf("[cache] EXPIRED %s", key)
		return nil, false
	}

	log.Printf("[cache] HIT     %s (expires in %s)", key, entry.expiresAt.Sub(now).Round(time.Second))
	return copyEvents(entry.events), true
}

// stores a successful result
func (c *EventCache) Set(key string, events []models.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := cacheEntry{events: copyEvents(events)}
	ttlText := "no expiry"
	if c.ttl > 0 {
		entry.expiresAt = c.now().Add(c.ttl)
		ttlText = "ttl " + c.ttl.String()
	}
	c.items[key] = entry
	log.Printf("[cache] STORE   %s (%d events, %s)", key, len(events), ttlText)
}

// removes every entry and returns how many were removed

func (c *EventCache) Clear() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := len(c.items)
	c.items = make(map[string]cacheEntry)
	log.Printf("[cache] CLEARED %d entries", n)
	return n
}

func (c *EventCache) DeleteByCity(city string) int {
	city = strings.ToLower(strings.TrimSpace(city))

	c.mu.Lock()
	defer c.mu.Unlock()

	n := 0
	for key := range c.items {
		parts := strings.Split(key, "|")
		if len(parts) == 3 && parts[0] == city {
			delete(c.items, key) // deleting while ranging is safe in Go
			n++
		}
	}
	log.Printf("[cache] DELETED %d entries for city %q", n, city)
	return n
}

//  removes every entry for a category (all cities).
func (c *EventCache) DeleteByCategory(category string) int {
	category = strings.ToLower(strings.TrimSpace(category))

	c.mu.Lock()
	defer c.mu.Unlock()

	n := 0
	for key := range c.items {
		parts := strings.Split(key, "|")
		if len(parts) == 3 && parts[2] == category {
			delete(c.items, key)
			n++
		}
	}
	log.Printf("[cache] DELETED %d entries for category %q", n, category)
	return n
}

//removes every entry for a country code
func (c *EventCache) DeleteByCountry(country string) int {
	country = strings.ToUpper(strings.TrimSpace(country))

	c.mu.Lock()
	defer c.mu.Unlock()

	n := 0
	for key := range c.items {
		parts := strings.Split(key, "|")
		if len(parts) == 3 && parts[1] == country {
			delete(c.items, key)
			n++
		}
	}
	log.Printf("[cache] DELETED %d entries for country %q", n, country)
	return n
}

func copyEvents(src []models.Event) []models.Event {
	out := make([]models.Event, len(src))
	copy(out, src)
	return out
}