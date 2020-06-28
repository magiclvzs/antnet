package antnet

type ArrayMap struct {
	rawCap   int32
	genArray []interface{}
	delArray []int32
	genIndex int32
	delIndex int32
}

func (r *ArrayMap) Clone() *ArrayMap {
	return &ArrayMap{
		rawCap:   r.rawCap,
		genArray: append([]interface{}{}, r.genArray...),
		delArray: append([]int32{}, r.delArray...),
		genIndex: r.genIndex,
		delIndex: r.delIndex,
	}
}

func NewArrayMap(cap int32) *ArrayMap {
	return &ArrayMap{
		rawCap:   cap,
		genArray: make([]interface{}, cap),
		delArray: make([]int32, cap),
		genIndex: -1,
		delIndex: -1,
	}
}

func (r *ArrayMap) add(value interface{}) int32 {
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

func (r *ArrayMap) del(index int32) {
	id := r.delIndex + 1
	r.delArray[id] = index
	r.genArray[index] = nil
	r.delIndex++
}

func (r *ArrayMap) set(key int32, value interface{}) {
	for key >= int32(len(r.genArray)) {
		newArray := make([]interface{}, r.rawCap)
		r.genArray = append(r.genArray, newArray...)
		newDelArray := make([]int32, r.rawCap)
		r.delArray = append(r.delArray, newDelArray...)
	}
	r.genArray[key] = value
}

func (r *ArrayMap) Add(value interface{}) int32 {
	return r.add(value)
}

func (r *ArrayMap) Set(key int32, value interface{}) {
	r.set(key, value)
}

func (r *ArrayMap) Del(index int32) {
	r.del(index)
}
func (r *ArrayMap) Len() int32 {
	return r.genIndex - r.delIndex
}
func (r *ArrayMap) RawLen() int32 {
	return int32(len(r.genArray))
}

func (r *ArrayMap) Get(index int32) interface{} {
	if index >= int32(len(r.genArray)) {
		return nil
	}
	return r.genArray[index]
}
