package vnix

import (
	"net/http"
)

type qHttpHandler struct {
	rootNode *QNode
}
func (this *qHttpHandler) ServeHTTP(hRes http.ResponseWriter, hReq *http.Request) {
	currentNode, handlerFunc, paramMap := this.rootNode.getMethodHandler(hReq.Method, hReq.URL.Path)

	if handlerFunc != nil {
		ctx := newContext(hReq, hRes, paramMap)
		handlerFunc(ctx)
		ctx.send()
		return
	}
	if currentNode != nil && currentNode.nodeType == vNodeFileServer {
		http.Error(hRes, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	if currentNode != nil {
		http.Error(hRes, "405 Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	http.Error(hRes, "404 Not Found", http.StatusNotFound)
}
