package vnix

import (
	"net/http"
)
// QHttpServer
type QHttpServer struct {
	*QNode
}
func New() *QHttpServer {
	return &QHttpServer{
		QNode: newNode("", vNodeStatic),
	}
}
func (this *QHttpServer) Run(address string) error {
	return http.ListenAndServe(address, &qHttpHandler{
		rootNode: this.QNode,
	})
}