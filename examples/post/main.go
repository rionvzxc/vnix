package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.POST("/form", func(ctx *vnix.QContext) {
		idText := ctx.Form("id", "")
		name := ctx.Form("name", "Unknown")
		id := vnix.TString(idText).Int(0)

		ctx.SetTextF("ID: %d\nName: %s", id, name)
	})
	app.Run(":8080")
}
