package main

import (
	"net/http"

	"github.com/rionvzxc/vnix"
)

type userParams struct {
	ID int `uri:"id"`
}
type userQuery struct {
	Term string   `query:"term"`
	Tags []string `query:"tag"`
}
type createUser struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}
type searchForm struct {
	Term  string `form:"term"`
	Limit int    `form:"limit"`
}
type requestHeaders struct {
	RequestID string `header:"X-Request-ID"`
	Accept    string `header:"Accept"`
}
type sessionCookie struct {
	Session string `cookie:"session"`
}
func main() {
	app := vnix.New()

	app.GET("/users/:id", func(ctx *vnix.QContext) {
		var params userParams
		var query userQuery

		if err := ctx.BindParam(&params); err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText(err.Error())
			return
		}
		if err := ctx.BindQuery(&query); err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText(err.Error())
			return
		}
		ctx.SetJSON(map[string]any{"id": params.ID, "term": query.Term, "tags": query.Tags})
	})

	app.POST("/users", func(ctx *vnix.QContext) {
		var input createUser

		if err := ctx.BindJSON(&input); err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText(err.Error())
			return
		}
		ctx.SetJSON(map[string]any{"name": input.Name, "age": input.Age})
	})

	app.POST("/search", func(ctx *vnix.QContext) {
		var form searchForm

		if err := ctx.BindForm(&form); err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText(err.Error())
			return
		}
		ctx.SetJSON(map[string]any{"term": form.Term, "limit": form.Limit})
	})

	app.GET("/headers", func(ctx *vnix.QContext) {
		var headers requestHeaders

		if err := ctx.BindHeader(&headers); err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText(err.Error())
			return
		}
		ctx.SetJSON(map[string]any{"request_id": headers.RequestID, "accept": headers.Accept})
	})

	app.GET("/session", func(ctx *vnix.QContext) {
		var cookie sessionCookie

		if err := ctx.BindCookie(&cookie); err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.SetText(err.Error())
			return
		}
		ctx.SetJSON(map[string]any{"session": cookie.Session})
	})

	app.Run(":8080")
}
