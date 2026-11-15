package main

import "github.com/rionvzxc/vnix"

func main() {
	app := vnix.New()

	app.GET("/items/:id", func(ctx *vnix.QContext) {
		ctx.SetJSON(vnix.TMap{
			"id":     ctx.Param("id", ""),
			"method": "GET",
		})
	})

	app.POST("/items", func(ctx *vnix.QContext) {
		var item struct {
			Name string `json:"name"`
		}
		if err := ctx.JSON(&item); err != nil {
			ctx.StatusCode(400)
			ctx.SetText("Invalid JSON request body")
			return
		}
		ctx.StatusCode(201)
		ctx.SetJSON(vnix.TMap{"name": item.Name})
	})

	app.PUT("/items/:id", func(ctx *vnix.QContext) {
		var item struct {
			Name string `json:"name"`
		}
		if err := ctx.JSON(&item); err != nil {
			ctx.StatusCode(400)
			ctx.SetText("Invalid JSON request body")
			return
		}
		ctx.SetJSON(vnix.TMap{
			"id":   ctx.Param("id", ""),
			"name": item.Name,
		})
	})

	app.PATCH("/items/:id", func(ctx *vnix.QContext) {
		ctx.SetJSON(vnix.TMap{
			"id":     ctx.Param("id", ""),
			"method": "PATCH",
		})
	})

	app.DELETE("/items/:id", func(ctx *vnix.QContext) {
		ctx.StatusCode(204)
	})

	app.Run(":8080")
}
