package routers

import (
	"Beego-event-explorer/controllers"

	web "github.com/beego/beego/v2/server/web"
)

func init() {
	// Pages (HTML)
	web.Router("/", &controllers.HomeController{}, "get:Index")
	web.Router("/events", &controllers.EventsController{}, "get:List")
	web.Router("/events/:eventId", &controllers.EventsController{}, "get:Details")

	// Ticket action and mock-mode destination
	web.Router("/redirect/:eventId", &controllers.RedirectController{}, "get:Go")
	web.Router("/demo/tickets/:eventId", &controllers.DemoController{}, "get:Ticket")

	// JSON APIs (autocomplete must come before the :placeId route)
	web.Router("/api/locations/autocomplete", &controllers.APIController{}, "get:Autocomplete")
	web.Router("/api/locations/:placeId", &controllers.APIController{}, "get:Place")

	// Cache maintenance
	web.Router("/delete", &controllers.CacheController{}, "get:DeleteAll")
	web.Router("/delete/:name", &controllers.CacheController{}, "get:DeleteByName")
}