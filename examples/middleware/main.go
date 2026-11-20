package main

import (
	"log"
	"time"

	"github.com/rionvzxc/vnix"
)

func logger(ctx *vnix.QContext) {
	start := time.Now()
	ctx.Next()
	log.Printf("request took %s", time.Since(start))
}

func identify(ctx *vnix.QContext) {
	ctx.SetHeader("X-Middleware", "default")
	ctx.Next()
}

func hello(ctx *vnix.QContext) {
	ctx.SetText("Hello with default middleware")
}

func main() {
	app := vnix.New()

	// Both handlers use the default Add mode and inherit to child routes.
	app.UseMiddleware(logger, identify)
	app.GET("/hello", hello)

	log.Fatal(app.Run(":8080"))
}
