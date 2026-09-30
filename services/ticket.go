package services

import (
	"errors"
	"net/url"
	"strings"
)

var ErrInvalidTicketURL = errors.New("ticket url is not allowed")

// Approved ticket hostnames. A host is allowed if it equals one of these
// or is a subdomain of one (www.ticketmaster.com, concerts.livenation.com, ...).
var DefaultTicketHosts = []string{
	// Ticketmaster country sites
	"ticketmaster.com",
	"ticketmaster.ca",
	"ticketmaster.co.uk",
	"ticketmaster.com.au",
	"ticketmaster.co.nz",
	"ticketmaster.com.mx",
	"ticketmaster.com.br",
	"ticketmaster.com.ar",
	"ticketmaster.cl",
	"ticketmaster.ie",
	"ticketmaster.de",
	"ticketmaster.es",
	"ticketmaster.nl",
	"ticketmaster.se",
	"ticketmaster.no",
	"ticketmaster.dk",
	"ticketmaster.fi",
	"ticketmaster.ae",
	"ticketmaster.at",
	"ticketmaster.be",
	"ticketmaster.ch",
	"ticketmaster.co.za",
	"ticketmaster.cz",
	"ticketmaster.pl",
	"ticketmaster.it",
	"ticketmaster.fr",

	// Ticketmaster's affiliate tracking domain, which the API returns for some
	// events. It forwards to the event's Ticketmaster page.
	"ticketmaster.evyy.net",

	// Ticketmaster group ticketing brands
	"livenation.com",
	"ticketweb.com",
	"universe.com",
	"frontgatetickets.com",
}

// ParseHostList turns "a.com, b.com" into ["a.com", "b.com"] (lowercased, trimmed).
func ParseHostList(csv string) []string {
	var out []string
	for _, h := range strings.Split(csv, ",") {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			out = append(out, h)
		}
	}
	return out
}

// ValidateTicketURL accepts only absolute https URLs on an approved host.
// It returns the cleaned URL string, or ErrInvalidTicketURL.
func ValidateTicketURL(raw string, allowed []string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Scheme != "https" || u.User != nil {
		return "", ErrInvalidTicketURL
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", ErrInvalidTicketURL
	}

	for _, d := range allowed {
		// The "." in the suffix matters: "evilticketmaster.com" must NOT match "ticketmaster.com".
		if host == d || strings.HasSuffix(host, "."+d) {
			return u.String(), nil
		}
	}
	return "", ErrInvalidTicketURL
}
