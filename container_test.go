package antnet

import (
	"testing"
)

func Test_MinHeap(t *testing.T) {
	mh := NewMinHeap()
	for i := 10000; i > 0; i-- {
		mh.Push(i, i)
	}
	m, p := mh.GetMin()
	top := mh.Top()
	Printf("min : %v %v %v\n", m, p, top)
	for i := 0; i < 10; i++ {
		x := mh.Pop()
		Printf("%v  ", x)
	}

	Println("")

	mh.Push(1, 554654)
	mh.Push(1, 333)
	i := mh.Pop()
	Println(i, mh.Len())

	mh.Push(0, 19384)
	i = mh.Pop()
	Println(i, mh.Len())
}

func Test_MaxHeap(t *testing.T) {
	mh := NewMaxHeap()
	for i := 10000; i > 0; i-- {
		mh.Push(i, i)
	}
	m, p := mh.GetMin()
	top := mh.Top()
	Printf("max : %v %v %v\n", m, p, top)
	for i := 0; i < 10; i++ {
		x := mh.Pop()
		Printf("%v  ", x)
	}

	Println("")

	mh.Push(1, 554654)
	mh.Push(1, 333)
	i := mh.Pop()
	Println(i, mh.Len())

	mh.Push(0, 19384)
	i = mh.Pop()
	Println(i, mh.Len())
}
