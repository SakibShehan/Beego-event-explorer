package services

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"Beego-event-explorer/models"
)

// runs once for the whole services package
func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}


// Helpers


// fakeClock lets a test move time forward instantly instead of sleeping.
type fakeClock struct{ now time.Time }

func (f *fakeClock) Now() time.Time          { return f.now }
func (f *fakeClock) Advance(d time.Duration) { f.now = f.now.Add(d) }

// newTestCache returns a cache wired to a fake clock.
func newTestCache(ttl time.Duration) (*EventCache, *fakeClock) {
	clock := &fakeClock{now: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}
	c := NewEventCache(ttl)
	c.now = clock.Now
	return c, clock
}

func sampleEvents(names ...string) []models.Event {
	out := make([]models.Event, 0, len(names))
	for i, n := range names {
		out = append(out, models.Event{ID: fmt.Sprintf("id-%d", i+1), Name: n})
	}
	return out
}

func eventNames(events []models.Event) []string {
	names := make([]string, 0, len(events))
	for _, e := range events {
		names = append(names, e.Name)
	}
	return names
}

func cacheKeys(c *EventCache) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	keys := make([]string, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func hasKey(c *EventCache, key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.items[key]
	return ok
}

// captureLog redirects the standard logger into a buffer for one test.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	return &buf
}

// normalizeSpaces collapses runs of whitespace so log checks do not depend on padding.
func normalizeSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

const (
	keyTorontoMusic    = "toronto|CA|music"
	keyTorontoSports   = "toronto|CA|sports"
	keyVancouverMusic  = "vancouver|CA|music"
	keyVancouverSports = "vancouver|CA|sports"
	keyChicagoMusic    = "chicago|US|music"
	keyChicagoSports   = "chicago|US|sports"
	keyLondonGBMusic   = "london|GB|music"
	keyLondonCAMusic   = "london|CA|music"
)

// holds 8 entries for chaching 
func seededCache(t *testing.T) *EventCache {
	t.Helper()
	c, _ := newTestCache(0)
	for _, key := range []string{
		keyTorontoMusic, keyTorontoSports,
		keyVancouverMusic, keyVancouverSports,
		keyChicagoMusic, keyChicagoSports,
		keyLondonGBMusic, keyLondonCAMusic,
	} {
		c.Set(key, sampleEvents("x"))
	}
	return c
}

// Construction and keys

func TestNewEventCache(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{"no expiry", 0},
		{"five minutes", 5 * time.Minute},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewEventCache(tc.ttl)
			if c == nil {
				t.Fatal("NewEventCache returned nil")
			}
			if c.ttl != tc.ttl {
				t.Errorf("ttl = %s, want %s", c.ttl, tc.ttl)
			}
			if c.items == nil || len(c.items) != 0 {
				t.Errorf("a new cache must start with an empty map, got %v", c.items)
			}
			if c.now == nil {
				t.Error("a new cache must have a clock")
			}
		})
	}
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name                    string
		city, country, category string
		want                    string
	}{
		{"already normalized", "toronto", "CA", "music", "toronto|CA|music"},
		{"city and category lower-cased, country upper-cased", "Toronto", "ca", "Music", "toronto|CA|music"},
		{"all caps input", "TORONTO", "Ca", "SPORTS", "toronto|CA|sports"},
		{"surrounding spaces are trimmed", "  Toronto ", " ca ", " Sports ", "toronto|CA|sports"},
		{"inner spaces in a city name are kept", "New York", "us", "Music", "new york|US|music"},
		{"non-ASCII city names are lower-cased", "SÃO PAULO", "br", "Music", "são paulo|BR|music"},
		{"empty parts still produce a key", "", "", "", "||"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CacheKey(tc.city, tc.country, tc.category); got != tc.want {
				t.Fatalf("CacheKey(%q, %q, %q) = %q, want %q", tc.city, tc.country, tc.category, got, tc.want)
			}
		})
	}
}




// Hits, misses and independence


func TestEventCache_MissThenHit(t *testing.T) {
	c, _ := newTestCache(5 * time.Minute)
	key := CacheKey("Toronto", "CA", "Music")
	want := sampleEvents("Jazz Night", "Rock Show")

	if got, ok := c.Get(key); ok || got != nil {
		t.Fatalf("an empty cache must miss, got ok=%v events=%v", ok, got)
	}

	c.Set(key, want)

	got, ok := c.Get(key)
	if !ok {
		t.Fatal("expected a hit after Set")
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("cached events = %v, want %v", got, want)
	}
}


func TestEventCache_EmptyListIsAValidEntry(t *testing.T) {
	tests := []struct {
		name   string
		events []models.Event
	}{
		{"nil slice", nil},
		{"empty slice", []models.Event{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestCache(0)
			c.Set("tinytown|CA|music", tc.events)

			got, ok := c.Get("tinytown|CA|music")
			if !ok {
				t.Fatal("a successful empty result must be cached, got a miss")
			}
			if len(got) != 0 {
				t.Fatalf("expected an empty list, got %v", got)
			}
		})
	}
}

func TestEventCache_ReturnsCopies(t *testing.T) {
	c, _ := newTestCache(0)
	key := CacheKey("Toronto", "CA", "Music")
	original := sampleEvents("A", "B")
	c.Set(key, original)

	original[0].Name = "CHANGED AFTER SET"
	first, _ := c.Get(key)
	if first[0].Name != "A" {
		t.Fatalf("the cache must not share the slice it was given, got %q", first[0].Name)
	}

	first[1].Name = "CHANGED AFTER GET"
	second, _ := c.Get(key)
	if second[1].Name != "B" {
		t.Fatalf("changing a returned slice must not change the cache, got %q", second[1].Name)
	}
}

// Expiry (fake clock, no sleeping)


func TestEventCache_Expiry(t *testing.T) {
	tests := []struct {
		name    string
		wait    time.Duration
		wantHit bool
	}{
		{"right after Set", 0, true},
		{"one second before expiry", 5*time.Minute - time.Second, true},
		{"exactly at expiry", 5 * time.Minute, false},
		{"one second after expiry", 5*time.Minute + time.Second, false},
		{"a day later", 24 * time.Hour, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, clock := newTestCache(5 * time.Minute)
			key := CacheKey("Toronto", "CA", "Music")
			c.Set(key, sampleEvents("A"))

			clock.Advance(tc.wait)

			_, ok := c.Get(key)
			if ok != tc.wantHit {
				t.Fatalf("after %s: hit = %v, want %v", tc.wait, ok, tc.wantHit)
			}
			if !tc.wantHit && hasKey(c, key) {
				t.Fatal("an expired entry must be removed from the map")
			}
		})
	}
}

func TestEventCache_NoExpiry(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{"ttl of zero", 0},
		{"negative ttl is treated as no expiry", -time.Minute},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, clock := newTestCache(tc.ttl)
			key := CacheKey("Toronto", "CA", "Music")
			c.Set(key, sampleEvents("A"))

			clock.Advance(10 * 365 * 24 * time.Hour) // ten years

			if _, ok := c.Get(key); !ok {
				t.Fatal("an entry without a ttl must never expire on its own")
			}

			c.mu.Lock()
			entry := c.items[key]
			c.mu.Unlock()
			if !entry.expiresAt.IsZero() {
				t.Fatalf("expiresAt should be unset, got %v", entry.expiresAt)
			}
		})
	}
}

// What the event service does: miss, fetch, Set. After expiry the entry is refreshed.
func TestEventCache_RefreshAfterExpiry(t *testing.T) {
	c, clock := newTestCache(5 * time.Minute)
	key := CacheKey("Toronto", "CA", "Music")

	c.Set(key, sampleEvents("old"))
	clock.Advance(6 * time.Minute)

	if _, ok := c.Get(key); ok {
		t.Fatal("the old entry should have expired")
	}

	c.Set(key, sampleEvents("fresh")) // the refreshed list

	got, ok := c.Get(key)
	if !ok || got[0].Name != "fresh" {
		t.Fatalf("expected the fresh list, got ok=%v events=%v", ok, got)
	}

	clock.Advance(4 * time.Minute)
	if _, ok := c.Get(key); !ok {
		t.Fatal("the refreshed entry should live for a full new window")
	}

	clock.Advance(2 * time.Minute) // now 6 minutes after the refresh
	if _, ok := c.Get(key); ok {
		t.Fatal("the refreshed entry should expire after its own window")
	}
}

func TestEventCache_SetRestartsTheClock(t *testing.T) {
	c, clock := newTestCache(5 * time.Minute)
	key := CacheKey("Toronto", "CA", "Music")

	c.Set(key, sampleEvents("first"))
	clock.Advance(4 * time.Minute)
	c.Set(key, sampleEvents("second")) // overwrite: the expiry restarts here

	clock.Advance(4 * time.Minute) // 8 minutes after the first Set, 4 after the second
	got, ok := c.Get(key)
	if !ok || got[0].Name != "second" {
		t.Fatalf("expected the overwritten entry to still be valid, got ok=%v events=%v", ok, got)
	}

	clock.Advance(2 * time.Minute) // 6 minutes after the second Set
	if _, ok := c.Get(key); ok {
		t.Fatal("the entry should expire 5 minutes after the latest Set")
	}
}

// The only test that uses the real clock, to prove the default clock works.
func TestEventCache_RealClockExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("sleeps for a short time")
	}
	c := NewEventCache(100 * time.Millisecond)
	key := CacheKey("Toronto", "CA", "Music")
	c.Set(key, sampleEvents("A"))

	if _, ok := c.Get(key); !ok {
		t.Fatal("expected a hit right after Set")
	}
	time.Sleep(150 * time.Millisecond)
	if _, ok := c.Get(key); ok {
		t.Fatal("expected a miss after the ttl has passed")
	}
}


// clear and the delete methods


func TestEventCache_Clear(t *testing.T) {
	c := seededCache(t)

	if got := c.Clear(); got != 8 {
		t.Fatalf("Clear() = %d, want 8", got)
	}
	if left := cacheKeys(c); len(left) != 0 {
		t.Fatalf("expected an empty cache, still has %v", left)
	}
	if _, ok := c.Get(keyTorontoMusic); ok {
		t.Fatal("expected a miss after Clear")
	}
	if got := c.Clear(); got != 0 {
		t.Fatalf("clearing an empty cache = %d, want 0", got)
	}

	c.Set(keyTorontoMusic, sampleEvents("A")) // still usable afterwards
	if _, ok := c.Get(keyTorontoMusic); !ok {
		t.Fatal("the cache must still work after Clear")
	}
}

type deleteCase struct {
	name        string
	arg         string
	wantDeleted int
	wantGone    []string
}

//seeds a fresh 8-entry cache for each case,
func runDeleteCases(t *testing.T, del func(*EventCache, string) int, cases []deleteCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := seededCache(t)
			before := len(cacheKeys(c))

			if got := del(c, tc.arg); got != tc.wantDeleted {
				t.Fatalf("deleted %d entries, want %d", got, tc.wantDeleted)
			}
			for _, key := range tc.wantGone {
				if hasKey(c, key) {
					t.Errorf("key %q should have been deleted", key)
				}
			}
			if left := len(cacheKeys(c)); left != before-tc.wantDeleted {
				t.Errorf("%d entries left, want %d (everything else must survive)", left, before-tc.wantDeleted)
			}
		})
	}
}

func TestEventCache_DeleteByCity(t *testing.T) {
	runDeleteCases(t, (*EventCache).DeleteByCity, []deleteCase{
		{"lower-case city", "toronto", 2, []string{keyTorontoMusic, keyTorontoSports}},
		{"city is matched case-insensitively", "Toronto", 2, []string{keyTorontoMusic, keyTorontoSports}},
		{"surrounding spaces are trimmed", "  toronto  ", 2, []string{keyTorontoMusic, keyTorontoSports}},
		{"same city name in two countries", "london", 2, []string{keyLondonGBMusic, keyLondonCAMusic}},
		{"city with no entries", "paris", 0, nil},
		{"empty name deletes nothing", "", 0, nil},
		{"a country code is not a city", "CA", 0, nil},
		{"a category is not a city", "music", 0, nil},
	})
}

func TestEventCache_DeleteByCategory(t *testing.T) {
	runDeleteCases(t, (*EventCache).DeleteByCategory, []deleteCase{
		{"music in every city", "music", 5,
			[]string{keyTorontoMusic, keyVancouverMusic, keyChicagoMusic, keyLondonGBMusic, keyLondonCAMusic}},
		{"category is matched case-insensitively", "Music", 5, []string{keyTorontoMusic}},
		{"surrounding spaces and caps", "  SPORTS ", 3,
			[]string{keyTorontoSports, keyVancouverSports, keyChicagoSports}},
		{"unknown category", "jazz", 0, nil},
		{"empty name deletes nothing", "", 0, nil},
		{"a city is not a category", "toronto", 0, nil},
	})
}

func TestEventCache_DeleteByCountry(t *testing.T) {
	runDeleteCases(t, (*EventCache).DeleteByCountry, []deleteCase{
		{"Canada", "CA", 5,
			[]string{keyTorontoMusic, keyTorontoSports, keyVancouverMusic, keyVancouverSports, keyLondonCAMusic}},
		{"code is matched case-insensitively", "ca", 5, []string{keyLondonCAMusic}},
		{"spaces are trimmed", " us ", 2, []string{keyChicagoMusic, keyChicagoSports}},
		{"United Kingdom keeps London, Canada", "gb", 1, []string{keyLondonGBMusic}},
		{"country with no entries", "ZZ", 0, nil},
		{"empty code deletes nothing", "", 0, nil},
		{"a city is not a country", "toronto", 0, nil},
	})
}

func TestEventCache_DeletesOnEmptyCache(t *testing.T) {
	tests := []struct {
		name string
		run  func(*EventCache) int
	}{
		{"Clear", func(c *EventCache) int { return c.Clear() }},
		{"DeleteByCity", func(c *EventCache) int { return c.DeleteByCity("toronto") }},
		{"DeleteByCategory", func(c *EventCache) int { return c.DeleteByCategory("music") }},
		{"DeleteByCountry", func(c *EventCache) int { return c.DeleteByCountry("CA") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestCache(0)
			if got := tc.run(c); got != 0 {
				t.Fatalf("deleted %d entries from an empty cache, want 0", got)
			}
		})
	}
}

func TestEventCache_DeleteSkipsMalformedKeys(t *testing.T) {
	c, _ := newTestCache(0)
	c.mu.Lock()
	c.items["justonepart"] = cacheEntry{}
	c.items["two|parts"] = cacheEntry{}
	c.items["a|b|c|d"] = cacheEntry{}
	c.mu.Unlock()

	tests := []struct {
		name string
		run  func() int
	}{
		{"city equal to a one-part key", func() int { return c.DeleteByCity("justonepart") }},
		{"city equal to the first part of a two-part key", func() int { return c.DeleteByCity("two") }},
		{"country equal to the second part of a two-part key", func() int { return c.DeleteByCountry("PARTS") }},
		{"category equal to the third part of a four-part key", func() int { return c.DeleteByCategory("c") }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.run(); got != 0 {
				t.Fatalf("deleted %d entries, want 0 (keys without exactly 3 parts are ignored)", got)
			}
		})
	}
	if n := len(cacheKeys(c)); n != 3 {
		t.Fatalf("all 3 malformed keys should remain, found %d", n)
	}
}

// Logging 


func TestEventCache_LogMessages(t *testing.T) {
	buf := captureLog(t)
	c, clock := newTestCache(5 * time.Minute)
	key := CacheKey("Toronto", "CA", "Music")

	steps := []struct {
		name string
		do   func()
		want string
	}{
		{"miss is logged", func() { c.Get(key) },
			"[cache] MISS toronto|CA|music"},
		{"store is logged with count and ttl", func() { c.Set(key, sampleEvents("A", "B")) },
			"[cache] STORE toronto|CA|music (2 events, ttl 5m0s)"},
		{"hit is logged with the time left", func() { clock.Advance(time.Minute); c.Get(key) },
			"[cache] HIT toronto|CA|music (expires in 4m0s)"},
		{"expiry is logged", func() { clock.Advance(10 * time.Minute); c.Get(key) },
			"[cache] EXPIRED toronto|CA|music"},
	}
	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			buf.Reset()
			s.do()
			if got := normalizeSpaces(buf.String()); !strings.Contains(got, s.want) {
				t.Fatalf("log output %q does not contain %q", got, s.want)
			}
		})
	}
}

func TestEventCache_LogMessages_NoExpiryAndDeletes(t *testing.T) {
	buf := captureLog(t)
	c, _ := newTestCache(0)
	key := CacheKey("Toronto", "CA", "Music")

	steps := []struct {
		name string
		do   func()
		want string
	}{
		{"store without a ttl", func() { c.Set(key, sampleEvents("A", "B")) },
			"[cache] STORE toronto|CA|music (2 events, no expiry)"},
		{"hit without a ttl", func() { c.Get(key) },
			"[cache] HIT toronto|CA|music (no expiry)"},
		{"delete by city", func() { c.DeleteByCity("Toronto") },
			`[cache] DELETED 1 entries for city "toronto"`},
		{"delete by category", func() { c.Set(key, sampleEvents("A")); c.DeleteByCategory("Music") },
			`[cache] DELETED 1 entries for category "music"`},
		{"delete by country", func() { c.Set(key, sampleEvents("A")); c.DeleteByCountry("ca") },
			`[cache] DELETED 1 entries for country "CA"`},
		{"clear", func() { c.Set(key, sampleEvents("A")); c.Clear() },
			"[cache] CLEARED 1 entries (whole cache)"},
	}
	for _, s := range steps {
		t.Run(s.name, func(t *testing.T) {
			buf.Reset()
			s.do()
			if got := normalizeSpaces(buf.String()); !strings.Contains(got, s.want) {
				t.Fatalf("log output %q does not contain %q", got, s.want)
			}
		})
	}
}


// Concurrency: the Music and Sports goroutines share one map


func TestEventCache_ConcurrentSetAndGet(t *testing.T) {
	c := NewEventCache(0)
	const workers = 50

	var wg sync.WaitGroup
	problems := make(chan string, workers)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := CacheKey(fmt.Sprintf("city%d", i), "CA", "Music")
			want := sampleEvents(fmt.Sprintf("event-%d", i))

			c.Set(key, want)
			got, ok := c.Get(key)
			if !ok || !reflect.DeepEqual(got, want) {
				problems <- fmt.Sprintf("worker %d: ok=%v got=%v want=%v", i, ok, got, want)
			}
		}(i)
	}
	wg.Wait()
	close(problems)

	for p := range problems {
		t.Error(p)
	}
	if n := len(cacheKeys(c)); n != workers {
		t.Fatalf("expected %d entries, found %d", workers, n)
	}
}

// Reads, writes, expiry and every delete at once
func TestEventCache_ConcurrentMixedOperations(t *testing.T) {
	c := NewEventCache(time.Millisecond) // tiny ttl, so expiry races with everything else
	const workers, rounds = 40, 200

	var wg sync.WaitGroup
	for g := 0; g < workers; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for j := 0; j < rounds; j++ {
				key := CacheKey(fmt.Sprintf("city%d", (g+j)%7), "CA", "Music")
				switch (g + j) % 7 {
				case 0, 1:
					c.Set(key, sampleEvents("A"))
				case 2, 3:
					c.Get(key)
				case 4:
					c.DeleteByCity(fmt.Sprintf("city%d", j%7))
				case 5:
					c.DeleteByCategory("music")
				case 6:
					if j%50 == 0 {
						c.Clear()
					} else {
						c.DeleteByCountry("CA")
					}
				}
			}
		}(g)
	}

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("cache operations did not finish: possible deadlock")
	}
}
