package services

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"Beego-event-explorer/models"
)

// fake can be used wherever the service expects a provider.
var _ EventProvider = (*fakeProvider)(nil)


// Fake provider: scripted answers, no network, no API keys


type fakeProvider struct {
	mu    sync.Mutex
	lists map[string][]models.Event // category -> events to return
	errs  map[string]error          // category -> error to return
	calls []string                  // "Category|city|country", in call order
	hook  func(category string)     // optional: runs inside ListEvents (used to block or signal)
}

func newFakeProvider() *fakeProvider {
	return &fakeProvider{
		lists: map[string][]models.Event{
			"Music":  sampleEvents("Jazz Night", "Rock Show"),
			"Sports": sampleEvents("Derby Day"),
		},
		errs: map[string]error{},
	}
}

func (f *fakeProvider) ListEvents(ctx context.Context, city, country, category string) ([]models.Event, error) {
	f.mu.Lock()
	f.calls = append(f.calls, category+"|"+city+"|"+country)
	hook := f.hook
	f.mu.Unlock()

	if hook != nil {
		hook(category)
	}
	if err := ctx.Err(); err != nil { // like the real client, respect cancellation
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if err := f.errs[category]; err != nil {
		return nil, err
	}
	return f.lists[category], nil
}

func (f *fakeProvider) GetEvent(context.Context, string) (models.Event, error) {
	return models.Event{}, errors.New("GetEvent is not used by these tests")
}

func (f *fakeProvider) setErr(category string, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.errs[category] = err
}

func (f *fakeProvider) callLog() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func (f *fakeProvider) totalCalls() int { return len(f.callLog()) }

func (f *fakeProvider) callCount(category string) int {
	n := 0
	for _, c := range f.callLog() {
		if strings.HasPrefix(c, category+"|") {
			n++
		}
	}
	return n
}

func newTestService(fp *fakeProvider, cache *EventCache) *EventService {
	return &EventService{Provider: fp, TicketHosts: DefaultTicketHosts, Cache: cache}
}

var (
	jazzAndRock = []string{"Jazz Night", "Rock Show"}
	derbyDay    = []string{"Derby Day"}
)

// Constructors


func TestNewEventService(t *testing.T) {
	fp := newFakeProvider()
	svc := NewEventService(fp)

	if svc.Provider != fp {
		t.Error("the service must use the provider it was given")
	}
	if !reflect.DeepEqual(svc.TicketHosts, DefaultTicketHosts) {
		t.Error("the service must start with the default approved ticket hosts")
	}
	if svc.Cache == nil {
		t.Fatal("the service must come with a cache")
	}
	if svc.Cache.ttl != 0 {
		t.Errorf("cache ttl = %s, want 0 (entries stay until deleted)", svc.Cache.ttl)
	}
}

// Listing: success and failure of each section


func TestEventService_Listing(t *testing.T) {
	errBoom := errors.New("boom")

	tests := []struct {
		name                        string
		music, sports               []models.Event
		musicErr, sportsErr         error
		wantMusic, wantSports       []string
		wantMusicErr, wantSportsErr bool
	}{
		{
			name:  "both succeed",
			music: sampleEvents(jazzAndRock...), sports: sampleEvents(derbyDay...),
			wantMusic: jazzAndRock, wantSports: derbyDay,
		},
		{
			name:  "empty lists are valid, not errors",
			music: []models.Event{}, sports: nil,
			wantMusic: []string{}, wantSports: []string{},
		},
		{
			name:  "music fails, the sports section is kept",
			sports: sampleEvents(derbyDay...), musicErr: errBoom,
			wantMusic: []string{}, wantSports: derbyDay, wantMusicErr: true,
		},
		{
			name:  "sports fails, the music section is kept",
			music: sampleEvents(jazzAndRock...), sportsErr: errBoom,
			wantMusic: jazzAndRock, wantSports: []string{}, wantSportsErr: true,
		},
		{
			name:     "both fail",
			musicErr: errBoom, sportsErr: errBoom,
			wantMusic: []string{}, wantSports: []string{}, wantMusicErr: true, wantSportsErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fp := newFakeProvider()
			fp.lists["Music"], fp.lists["Sports"] = tc.music, tc.sports
			fp.errs["Music"], fp.errs["Sports"] = tc.musicErr, tc.sportsErr
			svc := newTestService(fp, NewEventCache(0))

			res := svc.Listing(context.Background(), "Toronto", "CA")

			if got := eventNames(res.Music); !reflect.DeepEqual(got, tc.wantMusic) {
				t.Errorf("music = %v, want %v", got, tc.wantMusic)
			}
			if got := eventNames(res.Sports); !reflect.DeepEqual(got, tc.wantSports) {
				t.Errorf("sports = %v, want %v", got, tc.wantSports)
			}
			if (res.MusicErr != "") != tc.wantMusicErr {
				t.Errorf("music error message = %q, want an error: %v", res.MusicErr, tc.wantMusicErr)
			}
			if (res.SportsErr != "") != tc.wantSportsErr {
				t.Errorf("sports error message = %q, want an error: %v", res.SportsErr, tc.wantSportsErr)
			}
			if tc.wantMusicErr && !strings.Contains(res.MusicErr, "Music events are unavailable") {
				t.Errorf("unexpected music error text: %q", res.MusicErr)
			}
			if tc.wantSportsErr && !strings.Contains(res.SportsErr, "Sports events are unavailable") {
				t.Errorf("unexpected sports error text: %q", res.SportsErr)
			}
		})
	}
}

func TestEventService_Listing_AsksTheProviderForBothCategories(t *testing.T) {
	fp := newFakeProvider()
	svc := newTestService(fp, NewEventCache(0))

	svc.Listing(context.Background(), "Toronto", "CA")

	got := fp.callLog()
	sort.Strings(got) // the two goroutines may call in either order
	want := []string{"Music|Toronto|CA", "Sports|Toronto|CA"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("provider calls = %v, want %v", got, want)
	}
}

func TestEventService_Listing_CancelledRequest(t *testing.T) {
	fp := newFakeProvider()
	svc := newTestService(fp, NewEventCache(0))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res := svc.Listing(ctx, "Toronto", "CA")

	if res.MusicErr == "" || res.SportsErr == "" {
		t.Fatalf("a cancelled request must show both errors, got %+v", res)
	}
	if n := len(cacheKeys(svc.Cache)); n != 0 {
		t.Fatalf("nothing may be cached after a cancelled request, found %d entries", n)
	}
}


// Concurrency and channels


// Each fetch blocks until BOTH have started
func TestEventService_Listing_StartsBothFetchesBeforeEitherFinishes(t *testing.T) {
	fp := newFakeProvider()
	started := make(chan string, 2)
	release := make(chan struct{})
	var once sync.Once
	releaseAll := func() { once.Do(func() { close(release) }) }
	t.Cleanup(releaseAll)

	fp.hook = func(category string) {
		started <- category
		<-release
	}
	svc := newTestService(fp, NewEventCache(0))

	done := make(chan models.ListingResult, 1)
	go func() { done <- svc.Listing(context.Background(), "Toronto", "CA") }()

	seen := map[string]bool{}
	for len(seen) < 2 {
		select {
		case c := <-started:
			seen[c] = true
		case <-time.After(2 * time.Second):
			t.Fatalf("only %v started while the other fetch was blocked: the fetches are not concurrent", seen)
		}
	}
	releaseAll()

	select {
	case res := <-done:
		if !reflect.DeepEqual(eventNames(res.Music), jazzAndRock) || !reflect.DeepEqual(eventNames(res.Sports), derbyDay) {
			t.Fatalf("unexpected result: %+v", res)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Listing did not finish after both fetches were released")
	}
}

// The channel has a buffer of one, so a goroutine can hand over its result and
// exit even when nobody is receiving yet (no goroutine leak).
func TestEventService_fetchAsync_ResultWaitsInTheChannel(t *testing.T) {
	svc := newTestService(newFakeProvider(), nil)

	ch := svc.fetchAsync(context.Background(), "Toronto", "CA", "Music")

	deadline := time.Now().Add(2 * time.Second)
	for len(ch) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if len(ch) != 1 {
		t.Fatal("the goroutine should leave its result in the channel without a receiver waiting")
	}
	res := <-ch
	if res.Err != nil || !reflect.DeepEqual(eventNames(res.Events), jazzAndRock) {
		t.Fatalf("unexpected result: %+v", res)
	}
}


// Cache integration


func TestEventService_Listing_CacheReuse(t *testing.T) {
	fp := newFakeProvider()
	svc := newTestService(fp, NewEventCache(0))

	steps := []struct {
		name           string
		city, country  string
		wantTotalCalls int // provider calls so far, across all steps
	}{
		{"first search asks the provider for both categories", "Toronto", "CA", 2},
		{"the same search again is served from the cache", "Toronto", "CA", 2},
		{"city and country case do not matter", "toronto", "ca", 2},
		{"another city asks the provider again", "Vancouver", "CA", 4},
		{"same city name in another country is a separate entry", "Toronto", "US", 6},
		{"the first city is still cached", "Toronto", "CA", 6},
	}
	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			res := svc.Listing(context.Background(), st.city, st.country)

			if !reflect.DeepEqual(eventNames(res.Music), jazzAndRock) || !reflect.DeepEqual(eventNames(res.Sports), derbyDay) {
				t.Errorf("unexpected result: %+v", res)
			}
			if got := fp.totalCalls(); got != st.wantTotalCalls {
				t.Errorf("provider calls = %d, want %d", got, st.wantTotalCalls)
			}
		})
	}
	if n := len(cacheKeys(svc.Cache)); n != 6 {
		t.Fatalf("expected 6 cache entries (3 searches x 2 categories), found %d", n)
	}
}

func TestEventService_Listing_ErrorsAreNotCached(t *testing.T) {
	fp := newFakeProvider()
	fp.setErr("Music", errors.New("ticketmaster down"))
	svc := newTestService(fp, NewEventCache(0))

	first := svc.Listing(context.Background(), "Toronto", "CA")
	if first.MusicErr == "" {
		t.Fatal("expected a music error on the first search")
	}
	if hasKey(svc.Cache, keyTorontoMusic) {
		t.Error("a failed list must not be cached")
	}
	if !hasKey(svc.Cache, keyTorontoSports) {
		t.Error("the successful sports list should be cached")
	}

	fp.setErr("Music", nil) // Ticketmaster recovers
	second := svc.Listing(context.Background(), "Toronto", "CA")

	if second.MusicErr != "" || !reflect.DeepEqual(eventNames(second.Music), jazzAndRock) {
		t.Fatalf("the retry should succeed, got %+v", second)
	}
	if got := fp.callCount("Music"); got != 2 {
		t.Errorf("music provider calls = %d, want 2 (the failure was retried)", got)
	}
	if got := fp.callCount("Sports"); got != 1 {
		t.Errorf("sports provider calls = %d, want 1 (the second search came from the cache)", got)
	}
}

func TestEventService_Listing_WithoutCache(t *testing.T) {
	fp := newFakeProvider()
	svc := newTestService(fp, nil)

	for i := 1; i <= 2; i++ {
		res := svc.Listing(context.Background(), "Toronto", "CA")
		if !reflect.DeepEqual(eventNames(res.Music), jazzAndRock) {
			t.Fatalf("search %d: unexpected music %v", i, eventNames(res.Music))
		}
	}
	if got := fp.totalCalls(); got != 4 {
		t.Fatalf("provider calls = %d, want 4 (no cache, so every search calls the provider)", got)
	}
}

// cache hits logging
func TestEventService_Listing_Logging(t *testing.T) {
	svc := newTestService(newFakeProvider(), NewEventCache(0))

	steps := []struct {
		name      string
		want      []string
		forbidden []string
	}{
		{
			"first search is a miss and calls the provider",
			[]string{
				"[cache] MISS toronto|CA|music",
				"[events] Music: started for Toronto, CA",
				"[events] Sports: started for Toronto, CA",
				"[cache] STORE toronto|CA|music",
			},
			[]string{"[cache] HIT"},
		},
		{
			"second search is a hit and skips the provider",
			[]string{"[cache] HIT toronto|CA|music", "[cache] HIT toronto|CA|sports"},
			[]string{"started for", "[cache] MISS"},
		},
	}
	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			buf := captureLog(t)
			svc.Listing(context.Background(), "Toronto", "CA")
			out := normalizeSpaces(buf.String())

			for _, w := range st.want {
				if !strings.Contains(out, w) {
					t.Errorf("log should contain %q, got: %s", w, out)
				}
			}
			for _, f := range st.forbidden {
				if strings.Contains(out, f) {
					t.Errorf("log must not contain %q, got: %s", f, out)
				}
			}
		})
	}
}

// Ticket link check (uses the service's own approved host list)


func TestEventService_ValidTicketURL(t *testing.T) {
	tests := []struct {
		name       string
		extraHosts []string
		raw        string
		ok         bool
	}{
		{"a default host is allowed", nil, "https://www.ticketmaster.ca/event/1", true},
		{"an unknown host is rejected", nil, "https://partner.example.com/e", false},
		{"http is rejected", nil, "http://www.ticketmaster.com/e", false},
		{"an empty url is rejected", nil, "", false},
		{"a host added to the service is allowed", []string{"partner.example.com"}, "https://partner.example.com/e", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := newTestService(newFakeProvider(), nil)
			svc.TicketHosts = append(append([]string{}, DefaultTicketHosts...), tc.extraHosts...)

			got, err := svc.ValidTicketURL(tc.raw)

			if tc.ok {
				if err != nil || got != tc.raw {
					t.Fatalf("got (%q, %v), want (%q, nil)", got, err, tc.raw)
				}
				return
			}
			if !errors.Is(err, ErrInvalidTicketURL) {
				t.Fatalf("error = %v, want ErrInvalidTicketURL", err)
			}
		})
	}
}