package vnix

import (
	"net/http"
	"sort"
	"strings"
)

type qHttpHandler struct {
	rootNode *QNode
}
func (this *qHttpHandler) ServeHTTP(hRes http.ResponseWriter, hReq *http.Request) {
	currentNode, handlerFunc, paramMap, allowedMethods := this.rootNode.getMethodHandler(hReq.Method, hReq.URL.Path)

	if handlerFunc != nil {
		ctx := newContext(hReq, hRes, paramMap)
		ctx.handlers = append(currentNode.getMiddlewareHandlers(false), handlerFunc)
		ctx.Next()
		ctx.send()
		return
	}
	if len(allowedMethods) == 0 {
		http.Error(hRes, "404 Not Found", http.StatusNotFound)
		return
	}
	sort.Strings(allowedMethods)
	hRes.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	http.Error(hRes, "405 Method Not Allowed", http.StatusMethodNotAllowed)
}
