
package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()

	app.GET("/", func(ctx *vnix.QContext) {
		ctx.SetText("Welcome to VNIX!")
	})

	api := app.Group("/api")

	api.GET("/", func(ctx *vnix.QContext) {
		ctx.SetText("API Home")
	})
	api.GET("/users", func(ctx *vnix.QContext) {
		ctx.SetText("List of users")
	})
	api.GET("/products", func(ctx *vnix.QContext) {
		ctx.SetText("List of products")
	})
	app.Run(":8080")
}
