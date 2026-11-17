package mst

// item is a heap entry: Prim stores {weight, edge index},
// Dijkstra stores {distance, vertex index}.
type item struct {
	key int // what we minimize
	id  int // tie-breaker and payload
}

// minHeap implements container/heap's heap.Interface (Len, Less, Swap, Push, Pop).
// Use it through heap.Push / heap.Pop, never by calling Push/Pop directly.
type minHeap []item

func (h minHeap) Len() int { return len(h) }
func (h minHeap) Less(i, j int) bool {
	if h[i].key != h[j].key {
		return h[i].key < h[j].key
	}
	return h[i].id < h[j].id
}
func (h minHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

// Push and Pop need a pointer receiver because they change the slice length.
// heap.Interface uses `any`, so we convert with a type assertion: x.(item).
func (h *minHeap) Push(x any) { *h = append(*h, x.(item)) }
func (h *minHeap) Pop() any {
	old := *h
	last := old[len(old)-1]
	*h = old[:len(old)-1]
	return last
}
