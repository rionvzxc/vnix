package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.GET("/param/:id", func(ctx *vnix.QContext) {
		id := ctx.GetParam("id").Int(0)
		name := ctx.GetQuery("name").Str("Unknown")

		ctx.SetTextF("ID: %d\nName: %s", id, name)
	})
	app.Run(":8080")
}