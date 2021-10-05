package main

import (
	"github.com/jfixby/pin"
	"github.com/jfixby/pin/lang"
	"github.com/stevenle/topsort"
	"testing"
)


func TestTopoSort(t *testing.T) {
	// Initialize the graph.
	graph := topsort.NewGraph()

	// Add edges.
	graph.AddEdge("A", "B")
	graph.AddEdge("B", "C")

	// Topologically sort node A.
	r, e := graph.TopSort("A") // => [C, B, A]
	lang.CheckErr(e)
	pin.D("toposort", r)
}
