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

var timerMap map[string]*timeNode = make(map[string]*timeNode)
var timerMapLock sync.Mutex

type timeNode struct {
	Name     string
	Inteval  uint32
	Callback func(interface{}) uint32
	Args     interface{}
}

func SetTimeout(name string, inteval int, handler func(interface{}) uint32, args interface{}) {
	setTimeout(name, uint32(inteval), handler, args)
	LogInfo("new timerout:%v inteval:%v", name, inteval)
}

func setTimeout(name string, inteval uint32, handler func(interface{}) uint32, args interface{}) {
	if inteval <= 0 {
		return
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
		return
	}
	if inteval < basePerWheel[bucket]*offset {
		return
	}
	left -= basePerWheel[bucket] * (offset - 1)
	pos := (newest[bucket] + offset) % slotCntPerWheel[bucket]

	node := &timeNode{name, left, handler, args}

	timerMapLock.Lock()
	timerMap[name] = node
	timerMapLock.Unlock()

	timewheelsLock.Lock()
	timewheels[bucket][pos].PushBack(node)
	timewheelsLock.Unlock()
}

func DelTimeout(name string) {
	timerMapLock.Lock()
	if v, ok := timerMap[name]; ok {
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
			delete(timerMap, node.Name)
			timerMapLock.Unlock()

			if nil != node.Callback {
				Go(func() {
					if 0 == bucket || 0 == node.Inteval {
						t := node.Callback(node.Args)
						if t > 0 {
							setTimeout(node.Name, t, node.Callback, node.Args)
						}
					} else {
						setTimeout(node.Name, node.Inteval, node.Callback, node.Args)
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
