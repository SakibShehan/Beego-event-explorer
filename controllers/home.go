package controllers

import web "github.com/beego/beego/v2/server/web"

type HomeController struct {
	web.Controller
}

func (c *HomeController) Index() {
	c.Data["Title"] = "Event Explorer | Find your next one"
	c.Data["Script"] = "/static/js/home.js"
	c.TplName = "home.tpl"
}
