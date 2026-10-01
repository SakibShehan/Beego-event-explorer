package main

import (
	"log"
	"os"
	"strings"
	"time"

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

	// Caching
	if raw := strings.TrimSpace(os.Getenv("CACHE_TTL")); raw != "" {
		if ttl, err := time.ParseDuration(raw); err == nil && ttl > 0 {
			eventSvc.Cache = services.NewEventCache(ttl)
			log.Printf("[config] cache ttl: %s", ttl)
		} else {
			log.Printf("[config] ignoring invalid CACHE_TTL %q", raw)
		}
	} else {
		log.Printf("[config] cache: no automatic expiry")
	}

	//  approved ticket hosts from .env
	if extra := services.ParseHostList(os.Getenv("TICKET_ALLOWED_HOSTS")); len(extra) > 0 {
		eventSvc.TicketHosts = append(eventSvc.TicketHosts, extra...)
		log.Printf("[config] extra ticket hosts: %v", extra)
	}
	controllers.EventSvc = eventSvc

	// default ticket view mode is live
	mode := strings.ToLower(strings.TrimSpace(os.Getenv("TICKET_MODE")))
	if mode != "mock" {
		mode = "live"
	}
	controllers.TicketMode = mode
	log.Printf("[config] ticket mode: %s", mode)

	web.Run()
}