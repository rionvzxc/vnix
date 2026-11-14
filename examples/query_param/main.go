package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.GET("/param/:id", func(ctx *vnix.QContext) {
		id := ctx.Param("id").Int(0)
		name := ctx.Query("name").String("Unknown")

		ctx.SetTextF("ID: %d\nName: %s", id, name)
	})
	app.Run(":8080")
}
