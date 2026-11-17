package mst

// unionFind (disjoint-set union) tracks which vertices are already connected.
// It uses path compression and union by rank, so each operation is almost O(1).
type unionFind struct {
	parent []int
	rank   []int
}

func newUnionFind(n int) *unionFind {
	u := &unionFind{parent: make([]int, n), rank: make([]int, n)}
	for i := range u.parent {
		u.parent[i] = i // every vertex starts as its own component
	}
	return u
}

// find returns the root (representative) of x's component.
// Path compression: every vertex on the way is re-pointed straight at the root.
func (u *unionFind) find(x int) int {
	if u.parent[x] != x {
		u.parent[x] = u.find(u.parent[x])
	}
	return u.parent[x]
}

// union joins the components of a and b. It returns false if they were
// already in the same component (adding the edge would close a cycle).
func (u *unionFind) union(a, b int) bool {
	ra, rb := u.find(a), u.find(b)
	if ra == rb {
		return false
	}
	// Union by rank: hang the shorter tree under the taller one.
	switch {
	case u.rank[ra] < u.rank[rb]:
		u.parent[ra] = rb
	case u.rank[ra] > u.rank[rb]:
		u.parent[rb] = ra
	default:
		u.parent[rb] = ra
		u.rank[ra]++
	}
	return true
}

// roots returns the current component root of every vertex (for the UI colors).
func (u *unionFind) roots() []int {
	r := make([]int, len(u.parent))
	for i := range r {
		r[i] = u.find(i)
	}
	return r
}
