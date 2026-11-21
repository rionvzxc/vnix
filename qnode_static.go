package vnix

import "sort"

func (this *QNode) getStaticNode(name string) *QNode {
	index := sort.Search(len(this.staticNodes), func(i int) bool {
		return this.staticNodes[i].name >= name
	})
	if index < len(this.staticNodes) && this.staticNodes[index].name == name {
		return this.staticNodes[index]
	}
	return nil
}

// addStaticNode returns the existing or newly inserted child and whether it is new.
// Children stay sorted by name for binary search during request routing.
func (this *QNode) addStaticNode(name string) (*QNode, bool) {
	index := sort.Search(len(this.staticNodes), func(i int) bool {
		return this.staticNodes[i].name >= name
	})
	if index < len(this.staticNodes) && this.staticNodes[index].name == name {
		return this.staticNodes[index], false
	}
	node := newNode(name, vNodeTypeStatic)
	node.parentNode = this
	this.staticNodes = append(this.staticNodes, nil)
	copy(this.staticNodes[index+1:], this.staticNodes[index:])
	this.staticNodes[index] = node
	return node, true
}
