package vnix

import (
	"strings"
)

const (
	vNodeStatic TNodeType = iota
	vNodeParam
	vNodeWildcard
)

type QNode struct {
	name string
	nodeType TNodeType
	staticNodes []*QNode
	dynamicNode *QNode
	handlerFuncMap map[string]THandlerFunc
}
func newNode(name string, nodeType TNodeType) *QNode {
	return &QNode{
		name: name,
		nodeType: nodeType,
	}
}
func (this *QNode) GET(path string, handler THandlerFunc) *QNode {
	this.Handle("GET", path, handler)
	return this
}
func (this *QNode) POST(path string, handler THandlerFunc) *QNode {
	return this.Handle("POST", path, handler)
}
func (this *QNode) PUT(path string, handler THandlerFunc) *QNode {
	return this.Handle("PUT", path, handler)
}
func (this *QNode) DELETE(path string, handler THandlerFunc) *QNode {
	return this.Handle("PUT", path, handler)
}
func (this *QNode) PATCH(path string, handler THandlerFunc) *QNode {
	return this.Handle("PATCH", path, handler)
}
func (this *QNode) Handle(method string, path string, handler THandlerFunc) *QNode {
	lastNode := this.Group(path)
	lastNode.setMethodHandler(method, handler)
	return lastNode
}
func (this *QNode) Group(path string) *QNode {
	var part string
	currentNode := this

	for i := 0; i < len(path); i++ {
		if path[i] == '/' || i == len(path)-1 {
			if path[i] != '/' {
				part += string(path[i])
			}
			if part != "" {
				if part[0] == ':' {
					name := part[1:]

					if name == "" {
						panic("Invalid empty parameter name")
					}
					dynamicNode := currentNode.dynamicNode

					if dynamicNode != nil {
						if dynamicNode.nodeType == vNodeParam && dynamicNode.name == name {
							currentNode = dynamicNode
						} else {
							panic("Route parameter conflict")
						}
					} else {
						newParamNode := newNode(name, vNodeParam)
						currentNode.dynamicNode = newParamNode
						currentNode = newParamNode
					}
				} else if part[0] == '*' {
					if i < len(path)-1 {
						panic("Wildcard must be the last segment")
					}
					name := part[1:]

					if name == "" {
						name = "wildcard"
					}
					dynamicNode := currentNode.dynamicNode

					if dynamicNode != nil {
						if dynamicNode.nodeType == vNodeWildcard && dynamicNode.name == name {
							currentNode = dynamicNode
						} else {
							panic("Route wildcard conflict")
						}
					} else {
						newWildcardNode := newNode(name, vNodeWildcard)
						currentNode.dynamicNode = newWildcardNode
						currentNode = newWildcardNode
					}
					return currentNode
				} else {
					var foundNode *QNode

					for _, node := range currentNode.staticNodes {
						if node.name == part {
							foundNode = node
							break
						}
					}
					if foundNode != nil {
						currentNode = foundNode
					} else {
						newStaticNode := newNode(part, vNodeStatic)
						currentNode.staticNodes = append(currentNode.staticNodes, newStaticNode)
						currentNode = newStaticNode
					}
				}
				part = ""
			}
			continue
		}
		part += string(path[i])
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
			var foundStaticNode *QNode

			for _, node := range currentNode.staticNodes {
				if node.name == part {
					foundStaticNode = node
					break
				}
			}
			if foundStaticNode != nil {
				currentNode = foundStaticNode
			} else if currentNode.dynamicNode != nil {
				dynamicNode := currentNode.dynamicNode

				if dynamicNode.nodeType == vNodeParam {
					if paramMap == nil {
						paramMap = make(TValues)
					}
					paramMap[dynamicNode.name] = append(paramMap[dynamicNode.name], part)
					currentNode = dynamicNode
				} else if dynamicNode.nodeType == vNodeWildcard {
					if paramMap == nil {
						paramMap = make(TValues)
					}
					paramMap[dynamicNode.name] = append(paramMap[dynamicNode.name], path[start:])
					return currentNode, currentNode.handlerFuncMap[method], paramMap
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