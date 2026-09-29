package main

import (
	_ "Beego-event-explorer/routers"
	beego "github.com/beego/beego/v2/server/web"
)

func main() {
	beego.Run()
}

