package main

import (
	"log"

	"github.com/rionvzxc/vnix"
)

func rootMiddleware(ctx *vnix.QContext) {
	ctx.SetHeader("X-Root", "enabled")
	ctx.Next()
}

func apiMiddleware(ctx *vnix.QContext) {
	ctx.SetHeader("X-API", "enabled")
	ctx.Next()
}

func localMiddleware(ctx *vnix.QContext) {
	ctx.SetHeader("X-Local", "enabled")
	ctx.Next()
}

func apiHome(ctx *vnix.QContext) {
	ctx.SetText("API with local middleware")
}

func apiHello(ctx *vnix.QContext) {
	ctx.SetText("API with inherited middleware")
}

func publicHello(ctx *vnix.QContext) {
	ctx.SetText("Public without middleware")
}

func main() {
	app := vnix.New()
	app.UseMiddleware(rootMiddleware).ModeAdd().Inherit(true)

	api := app.Group("/api")
	api.UseMiddleware(apiMiddleware).ModeSet().Inherit(true)
	api.UseMiddleware(localMiddleware).ModeAdd().Inherit(false)
	api.GET("", apiHome)
	api.GET("/hello", apiHello)

	public := app.Group("/public")
	public.UseMiddleware(nil).ModeSet().Inherit(true)
	public.GET("/hello", publicHello)

	app.FileServer("/assets", "./examples/middleware2/assets").
		UseMiddleware(nil).ModeSet().Inherit(true)

	log.Fatal(app.Run(":8081"))
}
