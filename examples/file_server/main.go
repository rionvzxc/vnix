package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()

	app.GET("/", func(ctx *vnix.QContext) {
		ctx.SetText("Try /assets/hello.txt to view a static file.")
	})
	app.FileServer("/assets", "./examples/file_server/assets")

	app.Run(":8080")
}
