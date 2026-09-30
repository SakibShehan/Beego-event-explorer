package controllers

import (
	"errors"
	"log"
	"net/http"
	"strings"

	"Beego-event-explorer/services"

	web "github.com/beego/beego/v2/server/web"
)

// "live" -> View tickets goes to the real (validated) ticket URL from the API.
// "mock" -> View tickets goes to the local demo page.
// Set once in main.go from TICKET_MODE.
var TicketMode = "live"

func renderError(c *web.Controller, status int, heading, message string) {
	c.Ctx.Output.SetStatus(status)
	c.Data["Title"] = heading + " | Event Explorer"
	c.Data["Heading"] = heading
	c.Data["Message"] = message
	c.TplName = "error.tpl"
}

// lookupFailed turns an event lookup error into a 404 or a 502 page.
func lookupFailed(c *web.Controller, id string, err error) {
	if errors.Is(err, services.ErrEventNotFound) {
		renderError(c, 404, "We could not find that event",
			"It may have been removed or the link may be wrong.")
		return
	}
	log.Printf("[tickets] lookup %s failed: %v", id, err)
	renderError(c, 502, "Tickets are unavailable right now",
		"We could not reach the event provider. Please try again in a moment.")
}

// GET /redirect/:eventId  (a backend action, not a page)
type RedirectController struct {
	web.Controller
}

func (c *RedirectController) Go() {
	id := c.Ctx.Input.Param(":eventId")
	if !eventIDPattern.MatchString(id) {
		renderError(&c.Controller, 404, "We could not find that event",
			"That event link does not look right.")
		return
	}

	// Mock mode: the destination is our own demo page.
	if TicketMode == "mock" {
		c.Redirect("/demo/tickets/"+id, http.StatusFound)
		return
	}

	// Live mode: the destination is the event's own ticket link from the
	// Ticketmaster API. We look it up on the server, the visitor never supplies it.
	ev, err := EventSvc.Details(c.Ctx.Request.Context(), id)
	if err != nil {
		lookupFailed(&c.Controller, id, err)
		return
	}

	// The API returned no ticket link for this event.
	if strings.TrimSpace(ev.TicketURL) == "" {
		log.Printf("[tickets] %s has no ticket url in the API response", id)
		renderError(&c.Controller, 404, "No ticket page for this event",
			"Ticketmaster does not list an online ticket page for this event.")
		return
	}

	// The link must be HTTPS on an approved host before we redirect.
	dest, err := EventSvc.ValidTicketURL(ev.TicketURL)
	if err != nil {
		log.Printf("[tickets] BLOCKED %s: %v | url: %s", id, err, ev.TicketURL)
		renderError(&c.Controller, 502, "This ticket link is unavailable",
			"We could not verify a safe ticket link for this event.")
		return
	}

	log.Printf("[tickets] %s -> %s", id, dest)
	c.Redirect(dest, http.StatusFound)
}

// GET /demo/tickets/:eventId  (local ticket destination, mock mode only)
type DemoController struct {
	web.Controller
}

func (c *DemoController) Ticket() {
	id := c.Ctx.Input.Param(":eventId")
	if TicketMode != "mock" || !eventIDPattern.MatchString(id) {
		renderError(&c.Controller, 404, "Page not found",
			"This page is only available in demo mode.")
		return
	}

	ev, err := EventSvc.Details(c.Ctx.Request.Context(), id)
	if err != nil {
		lookupFailed(&c.Controller, id, err)
		return
	}

	c.Data["Title"] = "Your ticket | Event Explorer"
	c.Data["Event"] = ev
	c.Data["BackURL"] = "/events/" + id
	c.TplName = "demo_ticket.tpl"
}