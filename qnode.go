package vnix

import (
	"net/http"
	"path/filepath"
	"strings"
)

type tNodeType uint

const (
	vNodeTypeNone tNodeType = iota
	vNodeTypeStatic
	vNodeTypeParam
	vNodeTypeWildcard
	vNodeTypeFileServer
)

// QNode represents a route group and its registered HTTP handlers.
type QNode struct {
	name           string
	nodeType       tNodeType
	staticNodes    []*QNode
	dynamicNode    *QNode
	handlerFuncMap map[string]THandlerFunc
	parentNode     *QNode
	middleware     []*QMiddleware
}
func newNode(name string, nodeType tNodeType) *QNode {
	return &QNode{
		name:     name,
		nodeType: nodeType,
	}
}
// GET registers handler for GET requests at path.
func (this *QNode) GET(path string, handler THandlerFunc) *QNode {
	this.Handle("GET", path, handler)
	return this
}
// POST registers handler for POST requests at path.
func (this *QNode) POST(path string, handler THandlerFunc) *QNode {
	return this.Handle("POST", path, handler)
}
// PUT registers handler for PUT requests at path.
func (this *QNode) PUT(path string, handler THandlerFunc) *QNode {
	return this.Handle("PUT", path, handler)
}
// DELETE registers handler for DELETE requests at path.
func (this *QNode) DELETE(path string, handler THandlerFunc) *QNode {
	return this.Handle("DELETE", path, handler)
}
// PATCH registers handler for PATCH requests at path.
func (this *QNode) PATCH(path string, handler THandlerFunc) *QNode {
	return this.Handle("PATCH", path, handler)
}
// FileServer registers a file server at a route that ends with a static segment.
func (this *QNode) FileServer(routePath string, directory string) *QNode {
	lastNode := this.createChildNode(routePath, true, vNodeTypeFileServer)
	handler := func(ctx *QContext) {
		relativePath := filepath.FromSlash(ctx.Param("file_server_path", ""))

		if !filepath.IsLocal(relativePath) {
			ctx.StatusCode(http.StatusNotFound)
			ctx.SetText(http.StatusText(http.StatusNotFound))
			return
		}
		filePath := filepath.Join(directory, relativePath)

		if err := ctx.SetFile(filePath); err != nil {
			ctx.StatusCode(http.StatusNotFound)
			ctx.SetText(http.StatusText(http.StatusNotFound))
		}
	}
	lastNode.handlerFuncMap = map[string]THandlerFunc{
		http.MethodGet:  handler,
		http.MethodHead: handler,
	}
	return lastNode
}
// Handle registers handler for method at path.
func (this *QNode) Handle(method string, path string, handler THandlerFunc) *QNode {
	lastNode := this.Group(path)
	lastNode.setMethodHandler(method, handler)
	return lastNode
}
// Group returns the route group at path, creating missing route nodes.
func (this *QNode) Group(path string) *QNode {
	lastNode := this.createChildNode(path, false, vNodeTypeNone)
	return lastNode
}
func (this *QNode) createChildNode(path string, isStaticOnly bool, lastNodeType tNodeType) *QNode {
	var part string
	currentNode := this
	isNewNode := false

	for i := 0; i < len(path); i++ {
		if path[i] == '/' || i == len(path)-1 {
			if path[i] != '/' {
				part += path[i : i+1]
			}
			if part != "" {
				if currentNode.nodeType == vNodeTypeFileServer || currentNode.nodeType == vNodeTypeWildcard {
					panic("Cannot register a route below a file server or wildcard route")
				}
				isNewNode = false

				if part[0] == ':' {
					name := part[1:]

					if name == "" {
						panic("Invalid empty parameter name")
					}
					dynamicNode := currentNode.dynamicNode

					if dynamicNode != nil {
						if dynamicNode.nodeType == vNodeTypeParam && dynamicNode.name == name {
							currentNode = dynamicNode
						} else {
							panic("Route parameter conflict")
						}
					} else {
						newParamNode := newNode(name, vNodeTypeParam)
						newParamNode.parentNode = currentNode
						currentNode.dynamicNode = newParamNode
						currentNode = newParamNode
						isNewNode = true
					}
				} else if part[0] == '*' {
					if lastNodeType == vNodeTypeFileServer {
						panic("File server route must end with a static segment")
					}
					if i < len(path)-1 {
						panic("Wildcard must be the last segment")
					}
					name := part[1:]

					if name == "" {
						name = "wildcard"
					}
					dynamicNode := currentNode.dynamicNode

					if dynamicNode != nil {
						if dynamicNode.nodeType == vNodeTypeWildcard && dynamicNode.name == name {
							currentNode = dynamicNode
						} else {
							panic("Route wildcard conflict")
						}
					} else {
						newWildcardNode := newNode(name, vNodeTypeWildcard)
						newWildcardNode.parentNode = currentNode
						currentNode.dynamicNode = newWildcardNode
						currentNode = newWildcardNode
						isNewNode = true
					}
					return currentNode
				} else {
					currentNode, isNewNode = currentNode.addStaticNode(part)
				}
				part = ""
			}
			continue
		}
		part += path[i : i+1]
	}
	if lastNodeType == vNodeTypeFileServer {
		if !isStaticOnly {
			panic("File server node requires a static-only route")
		}
		if currentNode.nodeType == vNodeTypeParam || currentNode.nodeType == vNodeTypeWildcard {
			panic("File server route must end with a static segment")
		}
		if !isNewNode {
			if (currentNode.nodeType != vNodeTypeStatic && currentNode.nodeType != vNodeTypeNone) || len(currentNode.staticNodes) != 0 || currentNode.dynamicNode != nil || len(currentNode.handlerFuncMap) != 0 {
				panic("Static route conflicts with an existing route")
			}
		}
		currentNode.nodeType = vNodeTypeFileServer
	} else if isNewNode && currentNode.nodeType == vNodeTypeStatic {
		currentNode.nodeType = lastNodeType
	}
	return currentNode
}
func (this *QNode) getMethodHandler(method string, path string) (*QNode, THandlerFunc, TValues) {
	var paramMap TValues
	var start int
	currentNode := this

	for i := 0; i <= len(path); i++ {
		if i < len(path) && path[i] != '/' {
			continue
		}
		if start < i {
			part := path[start:i]

			if currentNode.nodeType == vNodeTypeFileServer {
				if paramMap == nil {
					paramMap = make(TValues)
				}
				paramMap["file_server_path"] = append(paramMap["file_server_path"], path[start:])
				return currentNode, currentNode.handlerFuncMap[method], paramMap
			}
			foundStaticNode := currentNode.getStaticNode(part)

			if foundStaticNode != nil {
				currentNode = foundStaticNode

				if currentNode.nodeType == vNodeTypeFileServer {
					if paramMap == nil {
						paramMap = make(TValues)
					}
					if i < len(path) {
						paramMap["file_server_path"] = append(paramMap["file_server_path"], path[i+1:])
					}
					return currentNode, currentNode.handlerFuncMap[method], paramMap
				}
			} else if currentNode.dynamicNode != nil {
				dynamicNode := currentNode.dynamicNode

				if dynamicNode.nodeType == vNodeTypeParam {
					if paramMap == nil {
						paramMap = make(TValues)
					}
					paramMap[dynamicNode.name] = append(paramMap[dynamicNode.name], part)
					currentNode = dynamicNode
				} else if dynamicNode.nodeType == vNodeTypeWildcard {
					if paramMap == nil {
						paramMap = make(TValues)
					}
					paramMap[dynamicNode.name] = append(paramMap[dynamicNode.name], path[start:])
					return dynamicNode, dynamicNode.handlerFuncMap[method], paramMap
				} else {
					return nil, nil, nil
				}
			} else {
				return nil, nil, nil
			}
		}
		start = i + 1
	}
	return currentNode, currentNode.handlerFuncMap[method], paramMap
}
func (this *QNode) setMethodHandler(method string, handlerFunc THandlerFunc) {
	method = strings.TrimSpace(method)
	method = strings.ToUpper(method)

	if method == "" {
		return
	}
	if this.handlerFuncMap == nil {
		this.handlerFuncMap = make(map[string]THandlerFunc)
	}
	this.handlerFuncMap[method] = handlerFunc
}

// UseMiddleware registers handlers and returns their configuration.
// The default mode is Add with inheritance enabled.
// Nil handlers are ignored. UseMiddleware(nil).ModeSet().Inherit(true)
// clears middleware for this node and its descendants.
// Configuration applies to existing and future descendants and must finish
// before serving requests.
func (this *QNode) UseMiddleware(handlers ...THandlerFunc) *QMiddleware {
	entry := &QMiddleware{inherit: true}

	for _, handler := range handlers {
		if handler != nil {
			entry.handlers = append(entry.handlers, handler)
		}
	}
	this.middleware = append(this.middleware, entry)
	return entry
}

func (this *QNode) getMiddlewareHandlers(inheritedOnly bool) []THandlerFunc {
	var handlers []THandlerFunc

	if this.parentNode != nil {
		handlers = this.parentNode.getMiddlewareHandlers(true)
	}
	for _, entry := range this.middleware {
		if inheritedOnly && !entry.inherit {
			continue
		}
		if entry.modeSet {
			handlers = nil
		}
		handlers = append(handlers, entry.handlers...)
	}
	return handlers
}
