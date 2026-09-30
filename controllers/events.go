package controllers

import (
	"regexp"
	"strings"

	"Beego-event-explorer/services"

	web "github.com/beego/beego/v2/server/web"
)

// Set once in main.go, after .env is loaded.
var EventSvc *services.EventService

var countryPattern = regexp.MustCompile(`^[A-Za-z]{2}$`)

type EventsController struct {
	web.Controller
}

func (c *EventsController) List() {
	city := strings.TrimSpace(c.GetString("city"))
	country := strings.ToUpper(strings.TrimSpace(c.GetString("countryCode")))

	if city == "" || len([]rune(city)) > 100 || !countryPattern.MatchString(country) {
		c.Ctx.Output.SetStatus(400)
		c.Data["Title"] = "Choose a city"
		c.Data["Heading"] = "Please choose a city first"
		c.Data["Message"] = "We need a city and a country to show events. Go back and select a city from the suggestions."
		c.TplName = "error.tpl"
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