package antnet

import (
	"math"
	"sort"
)

type rigibody struct {
	value  float64
	indexs []int
	Owner  interface{}
}

type collisionMgr struct {
	cap       int
	len       int
	delLen    int
	dirty     bool
	rigibodys []*rigibody
	dels      []*rigibody
}

func (r *collisionMgr) Len() int {
	if r.dirty {
		return r.len
	}
	return r.len - r.delLen
}
func (r *collisionMgr) Less(i, j int) bool {
	return r.rigibodys[i].value < r.rigibodys[j].value
}
func (r *collisionMgr) Swap(i, j int) {
	for x := 0; x < 2; x++ {
		if r.rigibodys[i].indexs[x] == i {
			r.rigibodys[i].indexs[x] = j
			break
		}
	}
	for x := 0; x < 2; x++ {
		if r.rigibodys[j].indexs[x] == j {
			r.rigibodys[j].indexs[x] = i
			break
		}
	}
	r.rigibodys[i], r.rigibodys[j] = r.rigibodys[j], r.rigibodys[i]
}

func (r *collisionMgr) Update(left, right float64, rb *rigibody){
	r.rigibodys[rb.indexs[0]].value = left
	r.rigibodys[rb.indexs[1]].value = right
}

func (r *collisionMgr) Add(left, right float64, owner interface{}) *rigibody {
	var rb1, rb2 *rigibody
	if r.delLen > 0 {
		rb1 = r.dels[r.delLen-1]
		rb2 = r.dels[r.delLen-2]
		r.delLen -= 2

		rb1.Owner = owner
		rb2.Owner = owner
		rb1.value = left
		rb2.value = right
	} else {
		if r.len+2 > r.cap {
			rigibodys := make([]*rigibody, r.cap*2)
			dels := make([]*rigibody, r.cap*2)
			for x := 0; x < r.len; x++ {
				rigibodys[x] = r.rigibodys[x]
				dels[x] = r.dels[x]
			}
			r.rigibodys = rigibodys
			r.dels = dels
			r.cap *= 2
		}
		indexs := make([]int, 2, 2)
		rb1 = &rigibody{value: left, Owner: owner, indexs: indexs}
		rb2 = &rigibody{value: right, Owner: owner, indexs: indexs}
		indexs[0] = r.len
		r.rigibodys[r.len] = rb1
		indexs[1] = r.len + 1
		r.rigibodys[r.len+1] = rb2
		r.len += 2
	}
	r.dirty = true
	return rb1
}

func (r *collisionMgr) Del(rb *rigibody) {
	for x := 0; x < 2; x++ {
		rigibody := r.rigibodys[rb.indexs[x]]
		rigibody.value = math.MaxFloat64
		r.dels[r.delLen+x] = rigibody
	}
	r.delLen += 2
	r.dirty = true
}

func (r *collisionMgr) GetCollision(rb *rigibody) (re []*rigibody) {
	var min, max, xx int
	if rb.indexs[0] > rb.indexs[1] {
		min = rb.indexs[1]
		for xx = min; xx >= 0; xx-- {
			if r.rigibodys[xx].value == r.rigibodys[min].value {
				min = xx
			} else {
				break
			}
		}

		max = rb.indexs[0]
		for xx = max; xx < r.len; xx++ {
			if r.rigibodys[xx].value == r.rigibodys[max].value {
				max = xx
			} else {
				break
			}
		}
	} else {
		min = rb.indexs[0]
		for xx = min; xx >= 0; xx-- {
			if r.rigibodys[xx].value == r.rigibodys[min].value {
				min = xx
			} else {
				break
			}
		}

		max = rb.indexs[1]
		for xx = max; xx < r.len; xx++ {
			if r.rigibodys[xx].value == r.rigibodys[max].value {
				max = xx
			} else {
				break
			}
		}
	}
	for x := min; x <= max; x++ {
		if rb.Owner != r.rigibodys[x].Owner {
			re = append(re, r.rigibodys[x])
		}
	}
	return
}

func (r *collisionMgr) Sort() {
	sort.Sort(r)
	r.dirty = false
}

func GetCollisionMgr(cap int) *collisionMgr {
	return &collisionMgr{
		cap:       cap,
		rigibodys: make([]*rigibody, cap),
		dels:      make([]*rigibody, cap),
	}
}
