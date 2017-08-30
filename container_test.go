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
	Println(i)

	mh.Push(1, 1)
	i = mh.Pop()
	Println(i)

	mh.Push(0, 19384)
	i = mh.Pop()
	Println(i)
}
