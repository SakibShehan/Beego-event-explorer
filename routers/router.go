package routers

import (
	"Beego-event-explorer/controllers"

	web "github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/", &controllers.HomeController{}, "get:Index")
	web.Router("/api/locations/autocomplete", &controllers.APIController{}, "get:Autocomplete")
	web.Router("/api/locations/:placeId", &controllers.APIController{}, "get:Place")
	web.Router("/events", &controllers.EventsController{}, "get:List")
	web.Router("/events/:eventId", &controllers.EventsController{}, "get:Details")

}
