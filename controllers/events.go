package controllers

import (
	"errors"
	"log"
	"net/url"
	"regexp"
	"strings"

	"Beego-event-explorer/services"

	web "github.com/beego/beego/v2/server/web"
)

// Set once in main file
var EventSvc *services.EventService

var (
	countryPattern = regexp.MustCompile(`^[A-Za-z]{2}$`)
	eventIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
)

type EventsController struct {
	web.Controller
}

// showError renders error.tpl 
func (c *EventsController) showError(status int, title, heading, message string) {
	c.Ctx.Output.SetStatus(status)
	c.Data["Title"] = title + " | Event Explorer"
	c.Data["Heading"] = heading
	c.Data["Message"] = message
	c.TplName = "error.tpl"
}

// get by eventes city and country code
func (c *EventsController) List() {
	city := strings.TrimSpace(c.GetString("city"))
	country := strings.ToUpper(strings.TrimSpace(c.GetString("countryCode")))

	if city == "" || len([]rune(city)) > 100 || !countryPattern.MatchString(country) {
		c.showError(400, "Choose a city", "Please choose a city first",
			"We need a city and a country to show events. Go back and select a city from the suggestions.")
		return
	}

	result := EventSvc.Listing(c.Ctx.Request.Context(), city, country)

	c.Data["Title"] = "Events in " + city + " | Event Explorer"
	c.Data["City"] = city
	c.Data["Country"] = country
	c.Data["Music"] = result.Music
	c.Data["Sports"] = result.Sports
	c.Data["MusicErr"] = result.MusicErr
	c.Data["SportsErr"] = result.SportsErr
	c.TplName = "listing.tpl"
}

// get by events id 
func (c *EventsController) Details() {
	id := c.Ctx.Input.Param(":eventId")

	if !eventIDPattern.MatchString(id) {
		c.showError(404, "Event not found", "We could not find that event",
			"That event link does not look right. Try searching for your city again.")
		return
	}

	ev, err := EventSvc.Details(c.Ctx.Request.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrEventNotFound) {
			c.showError(404, "Event not found", "We could not find that event",
				"It may have been removed or the link may be wrong. Try searching for your city again.")
			return
		}
		log.Printf("[events] details %s failed: %v", id, err)
		c.showError(502, "Something went wrong", "Event details are unavailable right now",
			"We could not reach the event provider. Please try again in a moment.")
		return
	}

	//Back to events
	backURL := "/"
	if ev.City != "" && ev.CountryCode != "" {
		q := url.Values{}
		q.Set("city", ev.City)
		q.Set("countryCode", ev.CountryCode)
		backURL = "/events?" + q.Encode()
	}

	c.Data["Title"] = ev.Name + " | Event Explorer"
	c.Data["Event"] = ev
	c.Data["BackURL"] = backURL
	c.TplName = "details.tpl"
}
