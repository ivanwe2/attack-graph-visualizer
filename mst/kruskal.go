package mst

import (
	"fmt"
	"sort"
)

// Kruskal builds a minimum spanning tree (or forest, if the graph is
// disconnected): scan the edges from lightest to heaviest and keep an edge
// only if it joins two different components.
// Complexity: O(E log E) for the sort; each union-find check is almost O(1).
func Kruskal(g Graph) Result {
	idx := g.indexByID()
	n := len(g.Nodes)

	order := make([]int, len(g.Edges))
	for i := range order {
		order[i] = i
	}
	// sort.Slice passes positions in `order`, so we compare order[a] with order[b].
	sort.Slice(order, func(a, b int) bool { return edgeLess(g.Edges, order[a], order[b]) })

	uf := newUnionFind(n)
	r := Result{TreeEdges: []int{}}
	r.Steps = append(r.Steps, Step{
		Kind:    "sort",
		Edge:    -1,
		Order:   order,
		Comp:    uf.roots(),
		Message: fmt.Sprintf("Сортираме %d ребра по тегло (при равни тегла – по реда на добавяне). Всеки връх е отделна компонента.", len(order)),
	})

	examined := 0
	for _, e := range order {
		if len(r.TreeEdges) == n-1 {
			break // a spanning tree on n vertices has exactly n-1 edges
		}
		examined++
		src, dst := g.Edges[e].Source, g.Edges[e].Target
		s := Step{Edge: e, Examined: examined, CycleChecks: examined}

		// One cycle check: are both ends already in the same component?
		if uf.union(idx[src], idx[dst]) {
			r.TreeEdges = append(r.TreeEdges, e)
			r.TotalWeight += g.Edges[e].Weight
			s.Kind = "accept"
			s.Message = fmt.Sprintf("Ребро %s: прието – свързва две различни компоненти.", g.name(e))
		} else {
			s.Kind = "reject"
			s.Message = fmt.Sprintf("Ребро %s: отхвърлено – ще образува цикъл (%s и %s вече са в една компонента).", g.name(e), src, dst)
		}
		s.Total = r.TotalWeight
		s.Comp = uf.roots()
		r.Steps = append(r.Steps, s)
	}

	r.Components = n - len(r.TreeEdges) // a forest of k trees on n vertices has n-k edges
	done := Step{
		Kind: "done", Edge: -1, Total: r.TotalWeight,
		Examined: examined, CycleChecks: examined, Comp: uf.roots(),
		Message: forestMessage(r, n),
	}
	if skipped := len(order) - examined; skipped > 0 {
		done.Message += fmt.Sprintf(" Останалите %d ребра не са разгледани – дървото вече има V-1 ребра.", skipped)
	}
	r.Steps = append(r.Steps, done)
	return r
}
