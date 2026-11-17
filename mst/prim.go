package mst

import (
	"container/heap"
	"fmt"
	"sort"
)

// Prim grows a tree from start, always adding the cheapest edge that leaves
// the tree. Candidate edges wait in a min-heap: O(E log V) overall.
//
// This is the "lazy" version: an edge stays in the heap even after both of its
// ends have joined the tree; when it is popped, the cycle check rejects it.
// If the graph is disconnected, a new tree is started from the next unvisited
// vertex, so the result is a minimum spanning forest.
func Prim(g Graph, start string) Result {
	n := len(g.Nodes)
	r := Result{TreeEdges: []int{}}
	if n == 0 {
		r.Steps = []Step{{Kind: "done", Edge: -1, Message: forestMessage(r, n)}}
		return r
	}

	idx := g.indexByID()
	adj := g.adjacency(idx)
	inTree := make([]bool, n)
	h := &minHeap{}
	examined, checks, visited := 0, 0, 0

	ends := func(e int) (int, int) { return idx[g.Edges[e].Source], idx[g.Edges[e].Target] }

	// add puts v into the tree and pushes its edges that lead outside the tree.
	add := func(v int) {
		inTree[v] = true
		visited++
		for _, e := range adj[v] {
			a, b := ends(e)
			if !inTree[a] || !inTree[b] {
				examined++
				heap.Push(h, item{key: g.Edges[e].Weight, id: e})
			}
		}
	}

	// frontier lists the heap edges that still cross from the tree to the
	// outside (the real candidates), cheapest first.
	frontier := func() []int {
		var f []int
		for _, it := range *h {
			a, b := ends(it.id)
			if inTree[a] != inTree[b] {
				f = append(f, it.id)
			}
		}
		sort.Slice(f, func(i, j int) bool { return edgeLess(g.Edges, f[i], f[j]) })
		return f
	}

	addStep := func(kind string, e int, vertex string, f []int, msg string) {
		r.Steps = append(r.Steps, Step{
			Kind: kind, Edge: e, Vertex: vertex, Message: msg, Frontier: f,
			Total: r.TotalWeight, Examined: examined, CycleChecks: checks,
		})
	}

	// Try start first, then every vertex in order. A vertex that is still
	// outside the tree at that point begins a new tree (disconnected graph).
	roots := []int{idx[start]}
	for v := range n { // Go 1.22+: range over an integer gives 0..n-1
		roots = append(roots, v)
	}

	for _, root := range roots {
		if inTree[root] {
			continue
		}
		r.Components++
		add(root)
		id := g.Nodes[root].ID
		if r.Components == 1 {
			addStep("visit", -1, id, frontier(), fmt.Sprintf("Започваме от връх %s. Ребрата от него стават кандидати.", id))
		} else {
			addStep("visit", -1, id, frontier(), fmt.Sprintf("Няма ребра, излизащи от дървото, но има необходени върхове – графът е несвързан. Започваме ново дърво от връх %s.", id))
		}

		for h.Len() > 0 && visited < n {
			f := frontier() // candidates before the choice, so the UI can show what was beaten
			e := heap.Pop(h).(item).id
			checks++
			a, b := ends(e)
			if inTree[a] && inTree[b] {
				addStep("reject", e, "", f, fmt.Sprintf("Ребро %s: отхвърлено – двата края вече са в дървото, ще образува цикъл.", g.name(e)))
				continue
			}
			v := b
			if inTree[b] {
				v = a
			}
			r.TreeEdges = append(r.TreeEdges, e)
			r.TotalWeight += g.Edges[e].Weight
			add(v)
			addStep("accept", e, g.Nodes[v].ID, f, fmt.Sprintf("Ребро %s: прието – най-лекото ребро, излизащо от дървото. Добавяме връх %s.", g.name(e), g.Nodes[v].ID))
		}
	}

	addStep("done", -1, "", nil, forestMessage(r, n))
	return r
}
