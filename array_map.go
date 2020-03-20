package antnet

import (
	"sync"
	"sync/atomic"
)

type arrayMap struct {
	rawCap   int
	safe     bool
	genArray []interface{}
	delArray []int32
	sync.Mutex
	genIndex int32
	delIndex int32
}

func NewArrayMap(cap int, threadSafe bool) *arrayMap {
	return &arrayMap{
		rawCap:   cap,
		safe:     threadSafe,
		genArray: make([]interface{}, cap),
		delArray: make([]int32, cap),
		genIndex: -1,
		delIndex: -1,
	}
}

func (r *arrayMap) addSafe(value interface{}) int32 {
	var id int32 = -1
	for r.delIndex > 0 && id == -1 {
		ov := r.delIndex
		nv := r.delIndex - 1
		if nv >= 0 {
			if atomic.CompareAndSwapInt32(&r.delIndex, ov, nv) {
				id = r.delArray[ov]
			}
		}
	}
	if id == -1 {
		id = atomic.AddInt32(&r.genIndex, 1)
		if id >= int32(len(r.genArray)) {
			r.Lock()
			if id >= int32(len(r.genArray)) {
				newArray := make([]interface{}, r.rawCap)
				r.genArray = append(r.genArray, newArray...)
				newDelArray := make([]int32, r.rawCap)
				r.delArray = append(r.delArray, newDelArray...)
			}
			r.Unlock()
		}
	}

	r.genArray[id] = value
	return id
}

func (r *arrayMap) delSafe(index int32) {
	id := atomic.AddInt32(&r.delIndex, 1)
	r.delArray[id] = index
}

func (r *arrayMap) Add(value interface{}) int32 {
	if r.safe {
		return r.addSafe(value)
	}
	return r.add(value)
}

func (r *arrayMap) Del(index int32) {
	if r.safe {
		r.delSafe(index)
	} else {
		r.del(index)
	}
}
func (r *arrayMap) Len() int32 {
	return r.genIndex - r.delIndex
}

func (r *arrayMap) Get(index int32) interface{} {
	return r.genArray[index]
}

func (r *arrayMap) add(value interface{}) int32 {
	var id int32 = -1
	if r.delIndex > 0 {
		id = r.delArray[r.delIndex]
		r.delIndex--
	}
	if id == -1 {
		id = r.genIndex + 1
		if id >= int32(len(r.genArray)) {
			newArray := make([]interface{}, r.rawCap)
			r.genArray = append(r.genArray, newArray...)
			newDelArray := make([]int32, r.rawCap)
			r.delArray = append(r.delArray, newDelArray...)
		}
	}
	r.genIndex++
	r.genArray[id] = value
	return id
}

func (r *arrayMap) del(index int32) {
	id := r.delIndex + 1
	r.delArray[id] = index
	r.delIndex++
}
