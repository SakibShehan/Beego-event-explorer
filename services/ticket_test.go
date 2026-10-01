package services

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// ValidateTicketURL: the security check behind /redirect/:eventId

func TestValidateTicketURL(t *testing.T) {
	hosts := []string{"ticketmaster.com", "ticketmaster.ca"}

	tests := []struct {
		name string
		raw  string
		want string // the URL we expect back; "" means it must be rejected
	}{
		// allowed host urls
		{"exact approved host", "https://ticketmaster.com/event/1", "https://ticketmaster.com/event/1"},
		{"subdomain of approved host", "https://www.ticketmaster.ca/event/abc?x=1#top", "https://www.ticketmaster.ca/event/abc?x=1#top"},
		{"deep subdomain", "https://a.b.ticketmaster.com/e", "https://a.b.ticketmaster.com/e"},
		{"uppercase host is accepted", "https://WWW.TICKETMASTER.COM/e", "https://WWW.TICKETMASTER.COM/e"},
		{"uppercase scheme is normalized", "HTTPS://www.ticketmaster.com/e", "https://www.ticketmaster.com/e"},
		{"explicit port is ignored", "https://www.ticketmaster.com:443/e", "https://www.ticketmaster.com:443/e"},
		{"surrounding spaces are trimmed", "  https://www.ticketmaster.com/e  ", "https://www.ticketmaster.com/e"},

		//  rejected
		{"http is rejected", "http://www.ticketmaster.com/e", ""},
		{"ftp is rejected", "ftp://www.ticketmaster.com/e", ""},
		{"javascript scheme is rejected", "javascript:alert(1)", ""},
		{"data scheme is rejected", "data:text/html,<h1>x</h1>", ""},

		// rejected: host tricks
		{"lookalike suffix is rejected", "https://evilticketmaster.com/e", ""},
		{"approved name inside another domain", "https://ticketmaster.com.evil.com/e", ""},
		{"approved name only in the path", "https://evil.com/ticketmaster.com", ""},
		{"approved name only in the query", "https://evil.com/?u=https://ticketmaster.com", ""},
		{"userinfo trick sends you to evil.com", "https://ticketmaster.com@evil.com/e", ""},
		{"credentials are rejected even on approved host", "https://user:pass@www.ticketmaster.com/e", ""},
		{"unapproved host", "https://www.example.com/e", ""},
		{"approved name with an unapproved tld", "https://ticketmaster.com.au/e", ""},

		// rejected: not an absolute URL 
		{"relative path", "/events/1", ""},
		{"protocol-relative url", "//evil.com/x", ""},
		{"empty string", "", ""},
		{"only spaces", "   ", ""},
		{"https without a host", "https:///event/1", ""},
		{"malformed url", "https://[::1", ""},
		{"space inside the host", "https://ticket master.com/e", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateTicketURL(tc.raw, hosts)

			if tc.want == "" {
				if err == nil {
					t.Fatalf("expected %q to be rejected, but it was allowed as %q", tc.raw, got)
				}
				if !errors.Is(err, ErrInvalidTicketURL) {
					t.Fatalf("expected ErrInvalidTicketURL, got %v", err)
				}
				if got != "" {
					t.Fatalf("a rejected url must return an empty string, got %q", got)
				}
				return
			}

			if err != nil {
				t.Fatalf("expected %q to be allowed, got error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("returned url = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestValidateTicketURL_NoApprovedHosts(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
	}{
		{"nil list", nil},
		{"empty list", []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateTicketURL("https://www.ticketmaster.com/e", tc.allowed)
			if !errors.Is(err, ErrInvalidTicketURL) {
				t.Fatalf("with no approved hosts everything must be rejected, got %v", err)
			}
		})
	}
}


// DefaultTicketHosts: the real approved list used by the app


func TestDefaultTicketHosts_RealWorldLinks(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		allowed bool
	}{
		{"ticketmaster.com event page", "https://www.ticketmaster.com/event/Z7r9jZ1A7abcd", true},
		{"ticketmaster.ca event page", "https://www.ticketmaster.ca/event/10006123ABCD", true},
		{"affiliate tracking host", "https://ticketmaster.evyy.net/c/252938/264167/4272?u=x", true},
		{"livenation subdomain", "https://concerts.livenation.com/event/x", true},
		{"ticketweb", "https://www.ticketweb.com/event/x", true},

		{"shared affiliate parent domain is not approved", "https://evyy.net/c/1", false},
		{"other affiliate customers are not approved", "https://someone-else.evyy.net/c/1", false},
		{"lookalike of an approved host", "https://www.ticketmaster.com.evil.net/e", false},
		{"http version of a real host", "http://www.ticketmaster.com/e", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateTicketURL(tc.raw, DefaultTicketHosts)
			if (err == nil) != tc.allowed {
				t.Fatalf("ValidateTicketURL(%q): allowed = %v, want %v", tc.raw, err == nil, tc.allowed)
			}
		})
	}
}

// Guards against typos 
func TestDefaultTicketHosts_Format(t *testing.T) {
	seen := make(map[string]bool)

	for _, h := range DefaultTicketHosts {
		t.Run(h, func(t *testing.T) {
			if h != strings.ToLower(h) {
				t.Errorf("host %q must be lower case", h)
			}
			if strings.ContainsAny(h, " /:@") {
				t.Errorf("host %q must be a bare domain (no scheme, path, port or spaces)", h)
			}
			if strings.HasPrefix(h, ".") || strings.HasSuffix(h, ".") {
				t.Errorf("host %q must not start or end with a dot", h)
			}
			if !strings.Contains(h, ".") {
				t.Errorf("host %q must contain a dot", h)
			}
			if seen[h] {
				t.Errorf("host %q is listed twice", h)
			}
			seen[h] = true
		})
	}
}


// ParseHostList: reads TICKET_ALLOWED_HOSTS from .env


func TestParseHostList(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"empty string", "", nil},
		{"single host", "a.com", []string{"a.com"}},
		{"two hosts", "a.com,b.com", []string{"a.com", "b.com"}},
		{"spaces around hosts", "  a.com ,   b.com  ", []string{"a.com", "b.com"}},
		{"upper case is lowered", "A.COM, B.Com", []string{"a.com", "b.com"}},
		{"empty items are skipped", "a.com,, ,b.com,", []string{"a.com", "b.com"}},
		{"only commas and spaces", " , ,, ", nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ParseHostList(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseHostList(%q) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}