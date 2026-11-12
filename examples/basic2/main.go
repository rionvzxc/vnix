
package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()

	app.GET("/", func(ctx *vnix.QContext) {
		ctx.SetText("Welcome to VNIX!")
	})
	app.GET("/hello", func(ctx *vnix.QContext) {
		ctx.SetText("Hello World!")
	})
	app.GET("/about", func(ctx *vnix.QContext) {
		ctx.SetText("This is a simple VNIX web server.")
	})
	app.Run(":8080")
}
