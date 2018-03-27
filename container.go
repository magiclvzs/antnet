package antnet

import (
	"container/heap"
)

type item struct {
	value    int // The value of the item; arbitrary.
	priority int // The priority of the item in the queue.
	index    int // The index of the item in the heap.
}

type priorityQueue struct {
	im map[int]int
	m  map[int]*item
	cf func(l, r *item) bool
}

func minHeapComp(l, r *item) bool { return l.priority < r.priority }
func maxHeapComp(l, r *item) bool { return l.priority > r.priority }

func (r *priorityQueue) Len() int           { return len(r.im) }
func (r *priorityQueue) Less(i, j int) bool { return r.cf(r.m[r.im[i]], r.m[r.im[j]]) }
func (r *priorityQueue) Swap(i, j int) {
	r.im[i], r.im[j] = r.im[j], r.im[i]
	r.m[r.im[i]].index = i
	r.m[r.im[j]].index = j
}

func (r *priorityQueue) Push(x interface{}) {
	it := x.(*item)
	if _, ok := r.m[it.value]; !ok {
		n := len(r.im)
		it.index = n
		r.im[n] = it.value
		r.m[it.value] = it
	} else {
		LogWarn("heap can't insert repeated value:%v", it.value)
	}
}

func (r *priorityQueue) Pop() interface{} {
	n := len(r.im)
	it := r.m[r.im[n-1]]
	delete(r.im, n-1)
	delete(r.m, it.value)
	return it
}

type Heap struct {
	p *priorityQueue
}

func (r *Heap) Push(priority, value int) {
	heap.Push(r.p, &item{priority: priority, value: value})
}

func (r *Heap) Pop() int {
	return heap.Pop(r.p).(*item).value
}

func (r *Heap) Update(value, priority int) {
	if it, ok := r.p.m[value]; ok {
		it.priority = priority
		heap.Fix(r.p, it.index)
	}
}

func (r *Heap) Top() int {
	_, v := r.GetMin()
	return v
}

func (r *Heap) GetMin() (priority int, value int) {
	it := heap.Pop(r.p).(*item)
	heap.Push(r.p, it)
	return it.priority, it.value
}

func (r *Heap) GetMax() (priority int, value int) {
	return r.GetMin()
}

func (r *Heap) GetPriority(value int) (priority int, find bool) {
	it, ok := r.p.m[value]
	return it.priority, ok
}

func (r *Heap) Len() int {
	return len(r.p.m)
}

func NewMinHeap() *Heap {
	h := &Heap{p: &priorityQueue{m: map[int]*item{}, im: map[int]int{}, cf: minHeapComp}}
	heap.Init(h.p)
	return h
}

func NewMaxHeap() *Heap {
	h := &Heap{p: &priorityQueue{m: map[int]*item{}, im: map[int]int{}, cf: maxHeapComp}}
	heap.Init(h.p)
	return h
}
