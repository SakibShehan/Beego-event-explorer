package routers

import (
	"Beego-event-explorer/controllers"

	web "github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/", &controllers.HomeController{}, "get:Index")
}