package vnix

import (
	"net/http"
)

type qHttpHandler struct {
	rootNode *QNode
}
func (this *qHttpHandler) ServeHTTP(hRes http.ResponseWriter, hReq *http.Request) {
	currentNode, handlerFunc, paramMap := this.rootNode.getMethodHandler(hReq.Method, hReq.URL.Path)

	if currentNode == nil {
		http.Error(hRes, "404 Not Found", http.StatusNotFound)
		return
	}
	if handlerFunc == nil {
		http.Error(hRes, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := newContext(hReq, hRes, paramMap)
	handlerFunc(ctx)
	ctx.send()
}