package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.GET("/", func(ctx *vnix.QContext) {
		ctx.SetText("Hello World!")
	})
	app.Run(":8080")
}