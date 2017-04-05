package antnet

import (
	"container/list"
	"sync"
	"time"
)

func timerTick() {
	for i := 0; i < wheelCnt; i++ {
		for j := uint32(0); j < slotCntPerWheel[i]; j++ {
			timewheels[i] = append(timewheels[i], list.New())
		}
	}
	StartTick = time.Now().UnixNano() / 1000000
	NowTick = StartTick
	Timestamp = NowTick / 1000
	Go(func() {
		for IsRuning() {
			Sleep(1)
			nowTick := NowTick
			NowTick = time.Now().UnixNano() / 1000000
			Timestamp = NowTick / 1000
			for nowTick < NowTick {
				nowTick++
				tick()
			}
		}
	})
}

func Sleep(ms int) {
	time.Sleep(time.Millisecond * time.Duration(ms))
}

const wheelCnt = 5

var slotCntPerWheel = [wheelCnt]uint32{256, 64, 64, 64, 64}
var rightShiftPerWheel = [wheelCnt]uint32{8, 6, 6, 6, 6}
var basePerWheel = [wheelCnt]uint32{1, 256, 256 * 64, 256 * 64 * 64, 256 * 64 * 64 * 64}

var newest [wheelCnt]uint32
var timewheels [5][]*list.List
var timewheelsLock sync.Mutex

var timerMap map[uint32]*timeNode = make(map[uint32]*timeNode)
var timerMapLock sync.Mutex
var timerIndex uint32 = 0

type timeNode struct {
	Id       uint32
	Inteval  uint32
	Callback func(...interface{}) uint32
	Args     []interface{}
}

func SetTimeout(inteval int, handler func(...interface{}) uint32, args ...interface{}) uint32 {
	id := setTimeout(0, uint32(inteval), handler, args)
	LogInfo("new timerout:%v inteval:%v", id, inteval)
	return id
}

func setTimeout(id uint32, inteval uint32, handler func(...interface{}) uint32, args ...interface{}) uint32 {
	if inteval <= 0 {
		return 0
	}
	bucket := 0
	offset := inteval
	left := inteval
	for offset >= slotCntPerWheel[bucket] {
		offset >>= rightShiftPerWheel[bucket]
		var tmp uint32 = 1
		if bucket == 0 {
			tmp = 0
		}
		left -= basePerWheel[bucket] * (slotCntPerWheel[bucket] - newest[bucket] - tmp)
		bucket++
	}
	if offset < 1 {
		return 0
	}
	if inteval < basePerWheel[bucket]*offset {
		return 0
	}
	left -= basePerWheel[bucket] * (offset - 1)
	pos := (newest[bucket] + offset) % slotCntPerWheel[bucket]
	node := &timeNode{id, left, handler, args}

	timerMapLock.Lock()
	if id > 0 {
		timerMap[id] = node
	} else {
		timerIndex++
		if timerIndex == 0 {
			timerIndex++
		}
		node.Id = timerIndex
		timerMap[timerIndex] = node
	}
	timerMapLock.Unlock()

	timewheelsLock.Lock()
	timewheels[bucket][pos].PushBack(node)
	timewheelsLock.Unlock()
	if id > 0 {
		return id
	}
	return timerIndex
}

func DelTimeout(index uint32) {
	timerMapLock.Lock()
	if v, ok := timerMap[index]; ok {
		v.Callback = nil
		v.Args = nil
	}
	timerMapLock.Unlock()
}

func tick() {
	var n *list.Element
	for bucket := 0; bucket < wheelCnt; bucket++ {
		newest[bucket] = (newest[bucket] + 1) % slotCntPerWheel[bucket] //当前指针递增1
		l := timewheels[bucket][newest[bucket]]

		timewheelsLock.Lock()
		for e := l.Front(); e != nil; e = n {
			node := e.Value.(*timeNode)

			timerMapLock.Lock()
			delete(timerMap, node.Id)
			timerMapLock.Unlock()

			if nil != node.Callback {
				Go(func() {
					if 0 == bucket || 0 == node.Inteval {
						t := node.Callback(node.Args)
						if t > 0 {
							setTimeout(node.Id, t, node.Callback, node.Args)
						}
					} else {
						setTimeout(node.Id, node.Inteval, node.Callback, node.Args)
					}
				})
			}

			n = e.Next()
			l.Remove(e)
		}
		timewheelsLock.Unlock()

		if 0 != newest[bucket] {
			break
		}
	}
}
