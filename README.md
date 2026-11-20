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
| [middleware](https://github.com/rionvzxc/vnix/tree/main/examples/middleware) | Middleware with default Add mode and inheritance |
| [middleware2](https://github.com/rionvzxc/vnix/tree/main/examples/middleware2) | Configuring middleware modes, inheritance, and file server exclusions |
| [post](https://github.com/rionvzxc/vnix/tree/main/examples/post) | Handling POST form data |
| [query_param](https://github.com/rionvzxc/vnix/tree/main/examples/query_param) | Working with query parameters and URL parameters |
| [methods](https://github.com/rionvzxc/vnix/tree/main/examples/methods) | Handling JSON requests with GET, POST, PUT, PATCH, and DELETE |
| [file_server](https://github.com/rionvzxc/vnix/tree/main/examples/file_server) | Serving files from a directory |
| [file_upload](https://github.com/rionvzxc/vnix/tree/main/examples/file_upload) | Receiving and saving a multipart file upload |
| [file_download](https://github.com/rionvzxc/vnix/tree/main/examples/file_download) | Sending a file as a browser download |
| [serve_file](https://github.com/rionvzxc/vnix/tree/main/examples/serve_file) | Serving a static file with an automatically detected content type |
| [serve_html](https://github.com/rionvzxc/vnix/tree/main/examples/serve_html) | Returning an HTML string with `SetHTML` |

## License

See [LICENSE](LICENSE) for details.
