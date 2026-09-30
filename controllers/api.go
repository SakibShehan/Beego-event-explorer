package controllers

import (
	"errors"
	"regexp"
	"strings"

	"Beego-event-explorer/services"

	web "github.com/beego/beego/v2/server/web"
)

// Set in main.go after .env is loaded.
var LocationSvc services.LocationProvider

var (
	sessionTokenPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,36}$`)
	placeIDPattern      = regexp.MustCompile(`^[A-Za-z0-9_-]{1,256}$`)
)

type APIController struct {
	web.Controller
}

func (c *APIController) jsonError(status int, msg string) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = map[string]string{"error": msg}
	c.ServeJSON()
}

// GET /api/locations/autocomplete?input=tor&sessionToken=abc
func (c *APIController) Autocomplete() {
	input := strings.TrimSpace(c.GetString("input"))
	token := c.GetString("sessionToken")

	if len([]rune(input)) < 3 {
		c.jsonError(400, "input must be at least 3 characters")
		return
	}
	if !sessionTokenPattern.MatchString(token) {
		c.jsonError(400, "sessionToken must be 1-36 letters, digits, underscores or hyphens")
		return
	}

	suggestions, err := LocationSvc.Autocomplete(c.Ctx.Request.Context(), input, token)
	if err != nil {
		c.jsonError(502, "city suggestions are unavailable right now")
		return
	}

	c.Data["json"] = map[string]any{"suggestions": suggestions}
	c.ServeJSON()
}

// GET /api/locations/:placeId?sessionToken=abc
func (c *APIController) Place() {
	placeID := c.Ctx.Input.Param(":placeId")
	token := c.GetString("sessionToken")

	if !placeIDPattern.MatchString(placeID) {
		c.jsonError(400, "invalid placeId")
		return
	}
	if !sessionTokenPattern.MatchString(token) {
		c.jsonError(400, "sessionToken must be 1-36 letters, digits, underscores or hyphens")
		return
	}

	loc, err := LocationSvc.PlaceCity(c.Ctx.Request.Context(), placeID, token)
	if err != nil {
		if errors.Is(err, services.ErrPlaceNotFound) {
			c.jsonError(404, "place not found")
			return
		}
		c.jsonError(502, "place lookup is unavailable right now")
		return
	}

	c.Data["json"] = loc
	c.ServeJSON()
}
