package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.POST("/upload", func(ctx *vnix.QContext) {
		file, err := ctx.File("file")

		if err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText("Could not read multipart upload")
			return
		}
		filename := filepath.Base(file.Filename())

		if filename == "." || filename == string(filepath.Separator) || filename == "" {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText("Missing multipart file field: file")
			return
		}
		if err := os.MkdirAll("uploads", 0755); err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			ctx.SetText("Could not prepare upload directory")
			return
		}
		if err := file.Save(filepath.Join("uploads", filename)); err != nil {
			ctx.StatusCode(http.StatusInternalServerError)
			ctx.SetText("Could not save uploaded file")
			return
		}
		ctx.SetText("Saved: " + filename)
	})

	app.Run(":8080")
}
