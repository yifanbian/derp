package main

import (
	"testing"

	"tailscale.com/types/key"
	"tailscale.com/util/set"
)

func newTestCache() *NodeCache {
	c := &NodeCache{nodes: map[string]set.Set[key.NodePublic]{}}
	empty := set.Set[key.NodePublic]{}
	c.merged.Store(&empty)
	return c
}

func TestNodeCacheMergesTailnets(t *testing.T) {
	a, b, other := key.NewNode().Public(), key.NewNode().Public(), key.NewNode().Public()
	c := newTestCache()

	if c.Contains(a) {
		t.Fatal("empty cache must deny")
	}

	c.update("t1", set.Of(a))
	c.update("t2", set.Of(b))
	if !c.Contains(a) || !c.Contains(b) {
		t.Fatal("keys from all tailnets must be present")
	}
	if c.Contains(other) {
		t.Fatal("unknown key must be denied")
	}

	c.update("t1", set.Of(other))
	if c.Contains(a) {
		t.Fatal("removed key must be denied after refresh")
	}
	if !c.Contains(b) || !c.Contains(other) {
		t.Fatal("other tailnets must be unaffected")
	}
}
