package main

import (
	"net/http"

	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.GET("/download", func(ctx *vnix.QContext) {
		if err := ctx.SetDownload("examples/file_download/report.txt"); err != nil {
			ctx.StatusCode(http.StatusNotFound)
			ctx.SetText("File not found")
			return
		}
	})
	app.Run(":8080")
}
