package postgres

import (
	"slices"
	"testing"
)

func TestTopoOrder(t *testing.T) {
	type node struct {
		id   int
		deps []int
	}
	nodes := []node{{1, []int{3}}, {2, nil}, {3, []int{2}}, {4, []int{4, 9}}, {5, []int{6}}, {6, []int{5}}}
	got := topoOrder(nodes, func(n node) int { return n.id }, func(n node) []int { return n.deps })
	ids := []int{}
	for _, n := range got {
		ids = append(ids, n.id)
	}
	if !slices.Equal(ids, []int{2, 3, 1, 4, 6, 5}) {
		t.Fatalf("order = %v", ids)
	}
}
