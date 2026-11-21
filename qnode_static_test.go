package vnix

import (
	"net/http"
	"reflect"
	"sort"
	"testing"
)

func TestAddStaticNodeKeepsChildrenSorted(t *testing.T) {
	for _, names := range [][]string{
		{"alpha", "beta", "gamma"},
		{"gamma", "beta", "alpha"},
		{"middle", "zebra", "alpha", "apple", "app", "Apple", "café", "日本", "😀"},
	} {
		root := newNode("", vNodeTypeStatic)
		var want []string
		for _, name := range names {
			node, isNew := root.addStaticNode(name)
			if !isNew || node.name != name || node.parentNode != root || node.nodeType != vNodeTypeStatic {
				t.Fatalf("unexpected inserted node for %q: %#v, new = %v", name, node, isNew)
			}
			want = append(want, name)
			sort.Strings(want)
			var got []string
			for _, child := range root.staticNodes {
				got = append(got, child.name)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("children = %v, want %v", got, want)
			}
			for _, child := range root.staticNodes {
				if root.getStaticNode(child.name) != child {
					t.Fatalf("lookup did not find %q", child.name)
				}
			}
		}
	}
}

func TestGetStaticNodeMissing(t *testing.T) {
	root := newNode("", vNodeTypeStatic)
	if root.getStaticNode("missing") != nil {
		t.Fatal("empty node should not match")
	}
	root.addStaticNode("beta")
	root.addStaticNode("delta")
	for _, name := range []string{"", "alpha", "charlie", "zebra", "bet", "beta2", "Beta"} {
		if root.getStaticNode(name) != nil {
			t.Fatalf("unexpected match for %q", name)
		}
	}
}

func TestAddStaticNodeReusesExistingNode(t *testing.T) {
	app := New()
	first := app.Group("/beta")
	first.UseMiddleware(func(ctx *QContext) { ctx.SetHeader("X-Node", "beta") })
	app.GET("/beta", func(ctx *QContext) { ctx.SetText("get") })
	app.POST("/beta", func(ctx *QContext) { ctx.SetText("post") })
	app.Group("/alpha")
	app.Group("/zebra")
	node, isNew := app.addStaticNode("beta")
	if isNew || node != first || len(app.staticNodes) != 3 {
		t.Fatal("duplicate registration must reuse the existing node")
	}
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		response := performRequest(&qHttpHandler{rootNode: app.QNode}, method, "/beta")
		if response.Code != http.StatusOK || response.Header().Get("X-Node") != "beta" {
			t.Fatalf("%s lost handler or middleware", method)
		}
	}
}
