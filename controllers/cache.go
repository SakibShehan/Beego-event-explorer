package controllers

import (
	"log"
	"strings"

	web "github.com/beego/beego/v2/server/web"
)

type CacheController struct {
	web.Controller
}

func (c *CacheController) reply(status int, body map[string]any) {
	c.Ctx.Output.SetStatus(status)
	c.Data["json"] = body
	c.ServeJSON()
}

// clears the whole cache
func (c *CacheController) DeleteAll() {
	if EventSvc == nil || EventSvc.Cache == nil {
		c.reply(503, map[string]any{"error": "cache is not available"})
		return
	}

	n := EventSvc.Cache.Clear()

	log.Printf("[delete] /delete -> removed %d entries (whole cache)", n)
	c.reply(200, map[string]any{"action": "delete_all", "deleted": n})
}


//   music or sports deletes that category ,anything else as a city name
func (c *CacheController) DeleteByName() {
	if EventSvc == nil || EventSvc.Cache == nil {
		c.reply(503, map[string]any{"error": "cache is not available"})
		return
	}

	name := strings.ToLower(strings.TrimSpace(c.Ctx.Input.Param(":name")))

	// "|" is the key separator, so it can never be part of a valid name.
	if name == "" || len([]rune(name)) > 100 || strings.Contains(name, "|") {
		c.reply(400, map[string]any{"error": "invalid name"})
		return
	}

	if name == "music" || name == "sports" {
		n := EventSvc.Cache.DeleteByCategory(name)
		log.Printf("[delete] /delete/%s -> removed %d %s entries (all cities)", name, n, name)
		c.reply(200, map[string]any{"action": "delete_category", "target": name, "deleted": n})
		return
	}

		if countryPattern.MatchString(name) {
		code := strings.ToUpper(name)
		n := EventSvc.Cache.DeleteByCountry(code)
		log.Printf("[delete] /delete/%s -> removed %d entries for country %s (all cities, music and sports)", name, n, code)
		c.reply(200, map[string]any{"action": "delete_country", "target": code, "deleted": n})
		return
	}

	n := EventSvc.Cache.DeleteByCity(name)
	log.Printf("[delete] /delete/%s -> removed %d entries for city %q (music and sports)", name, n, name)
	c.reply(200, map[string]any{"action": "delete_city", "target": name, "deleted": n})
}


