package main

import (
	"github.com/rionvzxc/vnix"
)

func main() {
	app := vnix.New()
	app.GET("/", func(ctx *vnix.QContext) {
		ctx.SetHTML(`<!doctype html>
<html lang="en">
<head>
    <meta charset="utf-8">
    <title>VNIX HTML example</title>
</head>
<body>
    <h1>Hello from VNIX</h1>
</body>
</html>`)
	})
	app.Run(":8080")
}
