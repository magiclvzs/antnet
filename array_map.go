package antnet

type ArrayMap struct {
	cap        int32
	rawCap     int32
	genArray   []interface{}
	delArray   []int32
	generArray []int32
	genIndex   int32
	delIndex   int32
}

func (r *ArrayMap) Clone() *ArrayMap {
	return &ArrayMap{
		cap:        r.cap,
		rawCap:     r.rawCap,
		genArray:   append([]interface{}{}, r.genArray...),
		delArray:   append([]int32{}, r.delArray...),
		generArray: append([]int32{}, r.generArray...),
		genIndex:   r.genIndex,
		delIndex:   r.delIndex,
	}
}

func NewArrayMap(cap int32, fixedArray bool) *ArrayMap {
	m := &ArrayMap{
		cap:        cap,
		rawCap:     cap,
		genArray:   make([]interface{}, cap),
		delArray:   make([]int32, cap),
		generArray: make([]int32, cap),
		genIndex:   -1,
		delIndex:   -1,
	}
	if fixedArray {
		m.genIndex = cap - 1
	}
	return m
}

func (r *ArrayMap) Add(value interface{}) int32 {
	var index int32 = -1
	if r.delIndex >= 0 {
		index = r.delArray[r.delIndex]
		r.delIndex--
	}
	if index == -1 {
		index = r.genIndex + 1
		r.genIndex++
		if index >= r.cap {
			newArray := make([]interface{}, r.rawCap)
			r.genArray = append(r.genArray, newArray...)
			newDelArray := make([]int32, r.rawCap)
			r.delArray = append(r.delArray, newDelArray...)
			newDelArray = make([]int32, r.rawCap)
			r.generArray = append(r.generArray, newDelArray...)
			r.cap += r.rawCap
		}
	}
	r.genArray[index] = value
	return index + (r.generArray[index] << 16)
}

func (r *ArrayMap) Set(key int32, value interface{}) {
	index := key & 0x0000FFFF
	r.genArray[index] = value
}

func (r *ArrayMap) Del(key int32) {
	index := key & 0x0000FFFF
	r.delArray[r.delIndex+1] = index
	r.genArray[index] = nil
	r.generArray[index]++
	r.delIndex++
}
func (r *ArrayMap) RawLen() int32 {
	return r.genIndex + 1
}
func (r *ArrayMap) RawGet(key int32) interface{} {
	if key >= r.cap {
		return nil
	}
	return r.genArray[key]
}
func (r *ArrayMap) Get(key int32) interface{} {
	index := key & 0x0000FFFF
	gener := key >> 16
	if index >= r.cap {
		return nil
	}
	if gener != r.generArray[index] {
		return nil
	}

	return r.genArray[index]
}
