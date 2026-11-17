// Package mst builds minimum spanning trees (Kruskal, Prim) and a shortest-path
// tree (Dijkstra) and records every decision as a Step, so a UI can replay it.
package mst

import (
	"errors"
	"fmt"
)

// Size limits. Every step carries a snapshot of up to V entries (component
// roots, frontier, distances keyed by vertex ID) and a run has about V+E
// steps, so the response grows like (V+E)·V·len(ID). These limits keep the
// worst case at a few MB.
const (
	MaxNodes    = 100
	MaxEdges    = 1000
	MaxIDLength = 32 // bytes
)

// Node is a vertex. X and Y only remember where the UI drew it.
type Node struct {
	ID string  `json:"id"`
	X  float64 `json:"x"`
	Y  float64 `json:"y"`
}

// Edge is an undirected weighted edge between two vertex IDs.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Weight int    `json:"weight"`
}

// Graph is an undirected weighted graph. Edges are referred to by their index.
type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

// Step is one event of an algorithm run. Fields that do not apply to the
// algorithm are left empty; `omitempty` keeps them out of the JSON.
type Step struct {
	Kind        string `json:"kind"` // sort, visit, accept, reject, relax, skip, done
	Edge        int    `json:"edge"` // index into Graph.Edges, -1 if the step has no edge
	Vertex      string `json:"vertex,omitempty"`
	Message     string `json:"message"` // human-readable explanation (Bulgarian, shown in the UI log)
	Total       int    `json:"total"`   // weight of the tree so far
	Examined    int    `json:"examined"`
	CycleChecks int    `json:"cycleChecks"`

	Order    []int          `json:"order,omitempty"`    // Kruskal: edge indices in sorted order
	Comp     []int          `json:"comp,omitempty"`     // Kruskal: union-find root of every vertex
	Frontier []int          `json:"frontier,omitempty"` // Prim/Dijkstra: candidate edges
	Dist     map[string]int `json:"dist,omitempty"`     // Dijkstra: known distances
}

// Result is what an algorithm returns: the replay steps and the final tree.
type Result struct {
	Steps       []Step `json:"steps"`
	TreeEdges   []int  `json:"treeEdges"` // edge indices in the order they were chosen
	TotalWeight int    `json:"totalWeight"`
	Components  int    `json:"components"` // number of trees in the spanning forest
}

// Run validates the graph and runs the named algorithm.
// start is the start vertex for Prim and Dijkstra; empty means the first vertex.
func Run(g Graph, algorithm, start string) (Result, error) {
	if err := g.validate(); err != nil {
		return Result{}, err
	}
	if start == "" && len(g.Nodes) > 0 {
		start = g.Nodes[0].ID
	}
	if algorithm != "kruskal" && len(g.Nodes) > 0 {
		if _, ok := g.indexByID()[start]; !ok {
			return Result{}, fmt.Errorf("непознат начален връх %q", start)
		}
	}

	switch algorithm {
	case "kruskal":
		return Kruskal(g), nil
	case "prim":
		return Prim(g, start), nil
	case "dijkstra":
		for _, e := range g.Edges {
			if e.Weight < 0 {
				return Result{}, errors.New("алгоритъмът на Дейкстра изисква неотрицателни тегла")
			}
		}
		return Dijkstra(g, start), nil
	default:
		return Result{}, fmt.Errorf("непознат алгоритъм %q", algorithm)
	}
}

// validate checks that vertex IDs are unique and every edge uses known vertices.
func (g Graph) validate() error {
	if len(g.Nodes) > MaxNodes || len(g.Edges) > MaxEdges {
		return fmt.Errorf("графът е твърде голям (до %d върха и %d ребра)", MaxNodes, MaxEdges)
	}
	seen := make(map[string]bool, len(g.Nodes))
	for _, n := range g.Nodes {
		if n.ID == "" {
			return errors.New("връх без име")
		}
		if len(n.ID) > MaxIDLength {
			return fmt.Errorf("твърде дълго име на връх (най-много %d байта)", MaxIDLength)
		}
		if seen[n.ID] {
			return fmt.Errorf("повтарящ се връх %q", n.ID)
		}
		seen[n.ID] = true
	}
	for i, e := range g.Edges {
		if !seen[e.Source] || !seen[e.Target] {
			return fmt.Errorf("ребро %d сочи към непознат връх (%s-%s)", i, e.Source, e.Target)
		}
	}
	return nil
}

// indexByID maps every vertex ID to its position in g.Nodes.
func (g Graph) indexByID() map[string]int {
	idx := make(map[string]int, len(g.Nodes))
	for i, n := range g.Nodes {
		idx[n.ID] = i
	}
	return idx
}

// edgeLess is the tie-breaking rule used everywhere: lighter edge first,
// and for equal weights the edge that was added to the graph first.
func edgeLess(edges []Edge, a, b int) bool {
	if edges[a].Weight != edges[b].Weight {
		return edges[a].Weight < edges[b].Weight
	}
	return a < b
}

// name formats an edge for log messages, e.g. "A-B (тегло 4)".
func (g Graph) name(e int) string {
	return fmt.Sprintf("%s-%s (тегло %d)", g.Edges[e].Source, g.Edges[e].Target, g.Edges[e].Weight)
}

// adjacency returns, for every vertex index, the indices of the edges touching it.
func (g Graph) adjacency(idx map[string]int) [][]int {
	adj := make([][]int, len(g.Nodes))
	for i, e := range g.Edges {
		u, v := idx[e.Source], idx[e.Target]
		adj[u] = append(adj[u], i)
		if v != u {
			adj[v] = append(adj[v], i)
		}
	}
	return adj
}

// forestMessage is the final log line shared by Kruskal and Prim.
func forestMessage(r Result, n int) string {
	switch {
	case n == 0:
		return "Графът е празен."
	case r.Components == 1:
		return fmt.Sprintf("Готово: минимално покриващо дърво с %d ребра, общо тегло %d.", len(r.TreeEdges), r.TotalWeight)
	default:
		return fmt.Sprintf("Графът е несвързан: получена е минимална покриваща гора от %d дървета (%d ребра), общо тегло %d.",
			r.Components, len(r.TreeEdges), r.TotalWeight)
	}
}
