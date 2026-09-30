package main

import (
	"log"
	"os"
	"strings"

	"Beego-event-explorer/controllers"
	_ "Beego-event-explorer/routers"
	"Beego-event-explorer/services"

	web "github.com/beego/beego/v2/server/web"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	controllers.LocationSvc = services.NewGoogleClient(os.Getenv("GOOGLE_API_KEY"))

	tm := services.NewTicketmasterClient(os.Getenv("TICKETMASTER_API_KEY"))
	eventSvc := services.NewEventService(tm)

	// Extra approved ticket hosts from .env (comma separated), so a new host
	// can be approved without changing code.
	if extra := services.ParseHostList(os.Getenv("TICKET_ALLOWED_HOSTS")); len(extra) > 0 {
		eventSvc.TicketHosts = append(eventSvc.TicketHosts, extra...)
		log.Printf("[config] extra ticket hosts: %v", extra)
	}
	controllers.EventSvc = eventSvc

	// Live is the default. Set TICKET_MODE=mock only for the local demo page.
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("TICKET_MODE")))
	if mode != "mock" {
		mode = "live"
	}
	controllers.TicketMode = mode
	log.Printf("[config] ticket mode: %s", mode)

	web.Run()
}