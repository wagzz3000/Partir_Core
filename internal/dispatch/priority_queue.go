package dispatch

import (
	"container/heap"
	"sync"
)

// PriorityItem wraps a ticket with priority metadata
type PriorityItem struct {
	TicketID string
	Priority int // Higher = more urgent
	Index    int // Managed by heap
}

// PriorityQueue implements a max-heap for ticket dispatch
type PriorityQueue struct {
	mu   sync.Mutex
	heap priorityHeap
}

// NewPriorityQueue creates a new priority queue
func NewPriorityQueue() *PriorityQueue {
	pq := &PriorityQueue{
		heap: make(priorityHeap, 0),
	}
	heap.Init(&pq.heap)
	return pq
}

// Push adds a ticket to the queue
func (pq *PriorityQueue) Push(item PriorityItem) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	heap.Push(&pq.heap, &item)
}

// Pop removes and returns the highest priority ticket
func (pq *PriorityQueue) Pop() *PriorityItem {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.heap.Len() == 0 {
		return nil
	}
	return heap.Pop(&pq.heap).(*PriorityItem)
}

// Len returns the queue length
func (pq *PriorityQueue) Len() int {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	return pq.heap.Len()
}

// --- heap.Interface implementation ---

type priorityHeap []*PriorityItem

func (h priorityHeap) Len() int           { return len(h) }
func (h priorityHeap) Less(i, j int) bool { return h[i].Priority > h[j].Priority } // Max-heap
func (h priorityHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].Index = i
	h[j].Index = j
}

func (h *priorityHeap) Push(x interface{}) {
	item := x.(*PriorityItem)
	item.Index = len(*h)
	*h = append(*h, item)
}

func (h *priorityHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.Index = -1
	*h = old[:n-1]
	return item
}
