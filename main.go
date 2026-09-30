package main

import (
	"os"

	"Beego-event-explorer/controllers"
	_ "Beego-event-explorer/routers"
	"Beego-event-explorer/services"

	web "github.com/beego/beego/v2/server/web"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	controllers.LocationSvc = services.NewGoogleClient(os.Getenv("GOOGLE_API_KEY"))

	web.Run()
}
