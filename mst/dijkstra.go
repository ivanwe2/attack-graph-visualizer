package mst

import (
	"container/heap"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Dijkstra builds the shortest-path tree from source: every reachable vertex
// is connected along a cheapest path from source. Weights must be >= 0.
// Complexity: O((V+E) log V) with a heap.
// Note: this tree is in general NOT a minimum spanning tree.
func Dijkstra(g Graph, source string) Result {
	n := len(g.Nodes)
	r := Result{TreeEdges: []int{}}
	if n == 0 {
		r.Steps = []Step{{Kind: "done", Edge: -1, Message: "Графът е празен."}}
		return r
	}

	idx := g.indexByID()
	adj := g.adjacency(idx)
	const inf = math.MaxInt
	dist := make([]int, n)
	parent := make([]int, n) // edge used to reach each vertex, -1 = none yet
	done := make([]bool, n)
	for i := range dist {
		dist[i] = inf
		parent[i] = -1
	}
	s := idx[source]
	dist[s] = 0
	h := &minHeap{{key: 0, id: s}}
	examined, checks := 0, 0

	other := func(e, v int) int {
		if idx[g.Edges[e].Source] == v {
			return idx[g.Edges[e].Target]
		}
		return idx[g.Edges[e].Source]
	}
	// frontier = tentative parent edges of vertices that are reached but not final.
	frontier := func() []int {
		var f []int
		for v := range n {
			if !done[v] && parent[v] >= 0 {
				f = append(f, parent[v])
			}
		}
		sort.Ints(f)
		return f
	}
	known := func() map[string]int {
		m := make(map[string]int)
		for v, d := range dist {
			if d != inf {
				m[g.Nodes[v].ID] = d
			}
		}
		return m
	}
	addStep := func(kind string, e int, vertex, msg string) {
		r.Steps = append(r.Steps, Step{
			Kind: kind, Edge: e, Vertex: vertex, Message: msg,
			Total: r.TotalWeight, Examined: examined, CycleChecks: checks,
			Frontier: frontier(), Dist: known(),
		})
	}

	for h.Len() > 0 {
		u := heap.Pop(h).(item).id
		checks++
		if done[u] {
			continue // an old heap entry: u was already finalized with a shorter distance
		}
		done[u] = true
		uid := g.Nodes[u].ID
		if parent[u] < 0 {
			addStep("visit", -1, uid, fmt.Sprintf("Започваме от връх %s с разстояние 0.", uid))
		} else {
			e := parent[u]
			r.TreeEdges = append(r.TreeEdges, e)
			r.TotalWeight += g.Edges[e].Weight
			addStep("accept", e, uid, fmt.Sprintf("Връх %s е окончателен: най-краткото разстояние е %d (през ребро %s).", uid, dist[u], g.name(e)))
		}

		for _, e := range adj[u] {
			v := other(e, u)
			if done[v] {
				continue
			}
			examined++
			vid := g.Nodes[v].ID
			nd := dist[u] + g.Edges[e].Weight
			switch {
			case dist[v] == inf:
				dist[v], parent[v] = nd, e
				heap.Push(h, item{key: nd, id: v})
				addStep("relax", e, vid, fmt.Sprintf("Ребро %s: достигаме %s за първи път, разстояние %d.", g.name(e), vid, nd))
			case nd < dist[v]:
				old := dist[v]
				dist[v], parent[v] = nd, e
				heap.Push(h, item{key: nd, id: v})
				addStep("relax", e, vid, fmt.Sprintf("Ребро %s: по-кратък път до %s – разстоянието намалява от %d на %d.", g.name(e), vid, old, nd))
			default:
				addStep("skip", e, vid, fmt.Sprintf("Ребро %s: не подобрява пътя до %s (%d ≥ %d).", g.name(e), vid, nd, dist[v]))
			}
		}
	}

	r.Components = n - len(r.TreeEdges)
	msg := fmt.Sprintf("Готово: дърво на най-кратките пътища от %s с %d ребра (сума на теглата %d). То обикновено НЕ е минимално покриващо дърво.",
		source, len(r.TreeEdges), r.TotalWeight)
	var unreachable []string
	for v := range n {
		if dist[v] == inf {
			unreachable = append(unreachable, g.Nodes[v].ID)
		}
	}
	if len(unreachable) > 0 {
		msg += " Графът е несвързан – недостижими върхове: " + strings.Join(unreachable, ", ") + "."
	}
	addStep("done", -1, "", msg)
	return r
}
