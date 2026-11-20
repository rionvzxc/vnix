package vnix

import (
	"net/http"
)

// QHttpServer provides HTTP routing and server startup.
type QHttpServer struct {
	// QNode exposes route registration methods on the server.
	*QNode
}

// New creates an HTTP server with an empty route tree.
func New() *QHttpServer {
	return &QHttpServer{
		QNode: newNode("", vNodeTypeStatic),
	}
}

// Run starts the HTTP server on address and returns any listen error.
func (this *QHttpServer) Run(address string) error {
	return http.ListenAndServe(address, &qHttpHandler{
		rootNode: this.QNode,
	})
}
