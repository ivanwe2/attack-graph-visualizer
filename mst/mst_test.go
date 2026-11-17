package mst

import (
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

// makeGraph builds a graph from space-separated vertex names and edges.
func makeGraph(names string, edges ...Edge) Graph {
	var g Graph
	for _, id := range strings.Fields(names) {
		g.Nodes = append(g.Nodes, Node{ID: id})
	}
	g.Edges = edges
	return g
}

// wikipediaGraph is the classic Kruskal example (MST weight 39).
func wikipediaGraph() Graph {
	return makeGraph("A B C D E F G",
		Edge{"A", "B", 7},  // 0
		Edge{"A", "D", 5},  // 1
		Edge{"B", "C", 8},  // 2
		Edge{"B", "D", 9},  // 3
		Edge{"B", "E", 7},  // 4
		Edge{"C", "E", 5},  // 5
		Edge{"D", "E", 15}, // 6
		Edge{"D", "F", 6},  // 7
		Edge{"E", "F", 8},  // 8
		Edge{"E", "G", 9},  // 9
		Edge{"F", "G", 11}, // 10
	)
}

func kinds(r Result) []string {
	var k []string
	for _, s := range r.Steps {
		k = append(k, s.Kind)
	}
	return k
}

func TestKruskalKnownGraph(t *testing.T) {
	r := Kruskal(wikipediaGraph())
	if r.TotalWeight != 39 || r.Components != 1 {
		t.Fatalf("got weight %d, components %d; want 39, 1", r.TotalWeight, r.Components)
	}
	// Sorted order: AD5, CE5, DF6, AB7, BE7, BC8(cycle), EF8(cycle), BD9(cycle), EG9.
	if want := []int{1, 5, 7, 0, 4, 9}; !reflect.DeepEqual(r.TreeEdges, want) {
		t.Errorf("tree edges = %v, want %v", r.TreeEdges, want)
	}
	last := r.Steps[len(r.Steps)-1]
	if last.Kind != "done" || last.Total != 39 || last.Examined != 9 {
		t.Errorf("last step = %+v", last)
	}
}

func TestPrimKnownGraph(t *testing.T) {
	r := Prim(wikipediaGraph(), "A")
	if r.TotalWeight != 39 || r.Components != 1 {
		t.Fatalf("got weight %d, components %d; want 39, 1", r.TotalWeight, r.Components)
	}
	// From A: AD5, DF6, AB7, BE7, CE5, then BC8, EF8, BD9 are rejected, then EG9.
	if want := []int{1, 7, 0, 4, 5, 9}; !reflect.DeepEqual(r.TreeEdges, want) {
		t.Errorf("tree edges = %v, want %v", r.TreeEdges, want)
	}
	want := []string{"visit", "accept", "accept", "accept", "accept", "accept", "reject", "reject", "reject", "accept", "done"}
	if got := kinds(r); !reflect.DeepEqual(got, want) {
		t.Errorf("step kinds = %v, want %v", got, want)
	}
}

func TestKruskalSteps(t *testing.T) {
	g := makeGraph("A B C D",
		Edge{"A", "B", 1}, Edge{"B", "C", 2}, Edge{"A", "C", 3}, Edge{"C", "D", 4})
	r := Kruskal(g)
	want := []string{"sort", "accept", "accept", "reject", "accept", "done"}
	if got := kinds(r); !reflect.DeepEqual(got, want) {
		t.Fatalf("step kinds = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(r.Steps[0].Order, []int{0, 1, 2, 3}) {
		t.Errorf("sort order = %v", r.Steps[0].Order)
	}
	if !strings.Contains(r.Steps[3].Message, "цикъл") {
		t.Errorf("reject message should mention the cycle: %q", r.Steps[3].Message)
	}
	// After A-B and B-C, A, B and C share one union-find root, D is alone.
	c := r.Steps[2].Comp
	if c[0] != c[1] || c[1] != c[2] || c[3] == c[0] {
		t.Errorf("components after two edges = %v", c)
	}
}

func TestTiesAreDeterministic(t *testing.T) {
	// A square with one diagonal, every weight equal: only edge order decides.
	g := makeGraph("A B C D",
		Edge{"A", "B", 1}, Edge{"B", "C", 1}, Edge{"C", "D", 1}, Edge{"D", "A", 1}, Edge{"A", "C", 1})
	if got := Kruskal(g).TreeEdges; !reflect.DeepEqual(got, []int{0, 1, 2}) {
		t.Errorf("Kruskal tree = %v, want [0 1 2]", got)
	}
	// From C the candidates are BC(1), CD(2), AC(4): the lowest index wins each time.
	if got := Prim(g, "C").TreeEdges; !reflect.DeepEqual(got, []int{1, 0, 2}) {
		t.Errorf("Prim tree = %v, want [1 0 2]", got)
	}
	// Running twice must give exactly the same steps.
	if !reflect.DeepEqual(Prim(g, "C"), Prim(g, "C")) || !reflect.DeepEqual(Kruskal(g), Kruskal(g)) {
		t.Error("runs are not deterministic")
	}
}

func TestDisconnectedGraphGivesForest(t *testing.T) {
	// Triangle A-B-C, edge D-E and an isolated vertex F: three trees.
	g := makeGraph("A B C D E F",
		Edge{"A", "B", 1}, Edge{"B", "C", 2}, Edge{"A", "C", 3}, Edge{"D", "E", 4})
	for name, r := range map[string]Result{"kruskal": Kruskal(g), "prim": Prim(g, "A")} {
		if r.TotalWeight != 7 || r.Components != 3 || len(r.TreeEdges) != 3 {
			t.Errorf("%s: weight %d, components %d, edges %d; want 7, 3, 3",
				name, r.TotalWeight, r.Components, len(r.TreeEdges))
		}
		if msg := r.Steps[len(r.Steps)-1].Message; !strings.Contains(msg, "гора") {
			t.Errorf("%s: final message should say it is a forest: %q", name, msg)
		}
	}
}

func TestEmptyAndSingleVertex(t *testing.T) {
	for _, g := range []Graph{{}, makeGraph("A")} {
		k, p := Kruskal(g), Prim(g, "A")
		if k.TotalWeight != 0 || p.TotalWeight != 0 || k.Components != len(g.Nodes) || p.Components != len(g.Nodes) {
			t.Errorf("graph %v: kruskal %+v, prim %+v", g, k, p)
		}
	}
}

func TestKruskalAndPrimAgreeOnRandomGraphs(t *testing.T) {
	rng := rand.New(rand.NewSource(42)) // fixed seed: the test is repeatable
	for round := 0; round < 300; round++ {
		g := randomGraph(rng, 1+rng.Intn(12), rng.Float64())
		start := g.Nodes[rng.Intn(len(g.Nodes))].ID
		k, p := Kruskal(g), Prim(g, start)

		if k.TotalWeight != p.TotalWeight || k.Components != p.Components {
			t.Fatalf("round %d: kruskal %d/%d, prim %d/%d (weight/components)\n%+v",
				round, k.TotalWeight, k.Components, p.TotalWeight, p.Components, g)
		}
		for _, r := range []Result{k, p} {
			if len(r.TreeEdges) != len(g.Nodes)-r.Components {
				t.Fatalf("round %d: %d edges for %d vertices and %d components", round, len(r.TreeEdges), len(g.Nodes), r.Components)
			}
			if !isForest(g, r.TreeEdges) {
				t.Fatalf("round %d: tree edges %v contain a cycle", round, r.TreeEdges)
			}
		}
	}
}

// randomGraph uses small weights (1-10) so that ties are common.
func randomGraph(rng *rand.Rand, n int, density float64) Graph {
	var g Graph
	for i := 0; i < n; i++ {
		g.Nodes = append(g.Nodes, Node{ID: string(rune('A' + i))})
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if rng.Float64() < density {
				g.Edges = append(g.Edges, Edge{g.Nodes[i].ID, g.Nodes[j].ID, 1 + rng.Intn(10)})
			}
		}
	}
	return g
}

// isForest checks with a fresh union-find that the edges contain no cycle.
func isForest(g Graph, edges []int) bool {
	idx := g.indexByID()
	uf := newUnionFind(len(g.Nodes))
	for _, e := range edges {
		if !uf.union(idx[g.Edges[e].Source], idx[g.Edges[e].Target]) {
			return false
		}
	}
	return true
}

func TestDijkstra(t *testing.T) {
	g := makeGraph("A B C D E",
		Edge{"A", "B", 4}, // 0
		Edge{"A", "C", 1}, // 1
		Edge{"C", "B", 2}, // 2
		Edge{"B", "D", 1}, // 3
		Edge{"C", "D", 5}, // 4
	) // E is unreachable
	r := Dijkstra(g, "A")
	if want := []int{1, 2, 3}; !reflect.DeepEqual(r.TreeEdges, want) {
		t.Errorf("tree edges = %v, want %v", r.TreeEdges, want)
	}
	last := r.Steps[len(r.Steps)-1]
	if want := map[string]int{"A": 0, "C": 1, "B": 3, "D": 4}; !reflect.DeepEqual(last.Dist, want) {
		t.Errorf("distances = %v, want %v", last.Dist, want)
	}
	if r.Components != 2 || !strings.Contains(last.Message, "E") {
		t.Errorf("components = %d, final message %q", r.Components, last.Message)
	}
}

func TestRunErrors(t *testing.T) {
	ok := makeGraph("A B", Edge{"A", "B", 1})
	tests := []struct {
		name      string
		g         Graph
		algorithm string
		start     string
	}{
		{"unknown algorithm", ok, "bfs", ""},
		{"unknown start", ok, "prim", "Z"},
		{"duplicate vertex", makeGraph("A A"), "kruskal", ""},
		{"edge to unknown vertex", makeGraph("A", Edge{"A", "Q", 1}), "kruskal", ""},
		{"negative weight for dijkstra", makeGraph("A B", Edge{"A", "B", -1}), "dijkstra", "A"},
		{"vertex id too long", makeGraph(strings.Repeat("x", MaxIDLength+1)), "kruskal", ""},
		{"too many vertices", randomGraph(rand.New(rand.NewSource(1)), MaxNodes+1, 0), "kruskal", ""},
	}
	for _, tc := range tests {
		if _, err := Run(tc.g, tc.algorithm, tc.start); err == nil {
			t.Errorf("%s: expected an error", tc.name)
		}
	}
	// Negative weights are fine for a spanning tree; empty start means the first vertex.
	r, err := Run(makeGraph("A B", Edge{"A", "B", -1}), "prim", "")
	if err != nil || r.TotalWeight != -1 {
		t.Errorf("prim with negative weight: %v, %+v", err, r)
	}
}
