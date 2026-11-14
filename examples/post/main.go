package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.POST("/form", func(ctx *vnix.QContext) {
		id := ctx.Form("id").Int(0)
		name := ctx.Form("name").String("Unknown")

		ctx.SetTextF("ID: %d\nName: %s", id, name)
	})
	app.Run(":8080")
}
