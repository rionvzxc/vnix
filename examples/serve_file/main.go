package main

import (
	"net/http"

	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.GET("/asset/js/main.js", func(ctx *vnix.QContext) {
		if err := ctx.SetFile("examples/serve_file/asset/js/main.js"); err != nil {
			ctx.StatusCode(http.StatusNotFound)
			ctx.SetText("File not found")
			return
		}
	})
	app.Run(":8080")
}
