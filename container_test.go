package antnet

import (
	"testing"
)

func Test_MinHeap(t *testing.T) {
	mh := NewMinHeap()
	for i := 10000; i > 0; i-- {
		mh.Push(i, i)
	}

	i := mh.Pop()
	Println(i, mh.Len())

	mh.Push(1, 554654)
	mh.Push(1, 333)
	i = mh.Pop()
	Println(i, mh.Len())

	mh.Push(0, 19384)
	i = mh.Pop()
	Println(i, mh.Len())
}
