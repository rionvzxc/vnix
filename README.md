# vnix

`vnix` is a lightweight and efficient web framework for Go, designed for building HTTP servers, REST APIs, and web applications. It provides a simple routing API and a context-based approach to handling HTTP requests and responses.

## Installation

```bash
go get github.com/rionvzxc/vnix
```

## Features

- Simple and lightweight API
- HTTP method routing: GET, POST, PUT, DELETE, PATCH
- Custom request and response handling
- Request binding for JSON, route parameters, query parameters, headers, forms, and cookies
- Directory serving with `FileServer`
- Context-based request handling with `QContext`
- HTTP server management
- Suitable for building REST APIs and web applications

## Quick Start

```go
package main

import (
    "github.com/rionvzxc/vnix"
)

func main() {
    app := vnix.New()

    app.GET("/", func(ctx *vnix.QContext) {
        ctx.SetText("Hello World!")
    })

    app.Run(":8080")
}
```

Run the application:

```bash
go run main.go
```

Open http://localhost:8080 to see the result.

## Examples

| Example | Description |
|---|---|
| [basic](https://github.com/rionvzxc/vnix/tree/main/examples/basic) | Basic HTTP server |
| [basic2](https://github.com/rionvzxc/vnix/tree/main/examples/basic2) | HTTP server with multiple routes |
| [basic_group](https://github.com/rionvzxc/vnix/tree/main/examples/basic_group) | Grouping routes with URL prefixes |
| [post](https://github.com/rionvzxc/vnix/tree/main/examples/post) | Handling POST form data |
| [query_param](https://github.com/rionvzxc/vnix/tree/main/examples/query_param) | Working with query parameters and URL parameters |
| [methods](https://github.com/rionvzxc/vnix/tree/main/examples/methods) | Handling JSON requests with GET, POST, PUT, PATCH, and DELETE |
| [binding](https://github.com/rionvzxc/vnix/tree/main/examples/binding) | Binding route params, queries, JSON, forms, headers, and cookies |
| [file_server](https://github.com/rionvzxc/vnix/tree/main/examples/file_server) | Serving files from a directory |
| [file_upload](https://github.com/rionvzxc/vnix/tree/main/examples/file_upload) | Receiving and saving a multipart file upload |
| [file_download](https://github.com/rionvzxc/vnix/tree/main/examples/file_download) | Sending a file as a browser download |
| [serve_file](https://github.com/rionvzxc/vnix/tree/main/examples/serve_file) | Serving a static file with an automatically detected content type |
| [serve_html](https://github.com/rionvzxc/vnix/tree/main/examples/serve_html) | Returning an HTML string with `SetHTML` |

## Request Binding

Binding methods fill a pointer to a struct and return parsing or conversion errors. Use the matching struct tag for each source: `uri`, `query`, `header`, `form`, `cookie`, or `json` with `BindParam`, `BindQuery`, `BindHeader`, `BindForm`, `BindCookie`, or `BindJSON`. Scalar fields use the first value; slice fields receive all values. `SetJSON` accepts any Go value that `encoding/json` can marshal.

```go
var input struct {
    ID   int    `uri:"id"`
    Name string `query:"name"`
}
if err := ctx.BindParam(&input); err != nil {
    ctx.StatusCode(http.StatusBadRequest)
    ctx.SetText(err.Error())
    return
}
if err := ctx.BindQuery(&input); err != nil {
    ctx.StatusCode(http.StatusBadRequest)
    ctx.SetText(err.Error())
    return
}
ctx.SetJSON(input)
```

## Static Directories

Register a directory with `app.FileServer("/assets", "./public")`. The final route segment must be static, while earlier segments may include parameters such as `/users/:id/photo`. Files are served for GET and HEAD requests below that route.

## Headers and Multi-Value Accessors

`THeader` stores HTTP header values under canonical header names and provides `Set`, `Add`, `Get`, `GetDefault`, `GetLast`, `GetAll`, `Exist`, `Del`, and `Clone`. `TValues` provides the same multi-value operations except for header-name canonicalization. `GetAll` returns a copy, and `Clone` copies the map and its slices.

`Header(name, defaultValue)`, `Query(name, defaultValue)`, `Param(name, defaultValue)`, `Cookie(name, defaultValue)`, and `Form(name, defaultValue)` return the first value or the supplied default. Their `...Values(name)` counterparts return copies of all values for one key. `RequestHeader()`, `RequestQuery()`, `RequestParam()`, `RequestCookie()`, and `RequestForm()` return cloned maps.

`SetHeader(name, values...)` replaces response values for one header, while `AddHeader(name, values...)` appends values. `SetResponseHeader(header)` replaces all response headers with a clone.

## License

See [LICENSE](LICENSE) for details.
