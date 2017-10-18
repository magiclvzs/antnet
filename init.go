package antnet

import (
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

type Statis struct {
	GoCount     int
	MsgqueCount int
	StartTime   time.Time
	LastPanic   int
	PanicCount  int32
}

type WaitGroup struct {
	count int64
}

func (r *WaitGroup) Add(delta int) {
	atomic.AddInt64(&r.count, int64(delta))
}

func (r *WaitGroup) Done() {
	atomic.AddInt64(&r.count, -1)
}

func (r *WaitGroup) Wait() {
	for atomic.LoadInt64(&r.count) > 0 {
		Sleep(1)
	}
}

func (r *WaitGroup) TryWait() bool {
	return atomic.LoadInt64(&r.count) == 0
}

var statis = Statis{}
var waitAll = &WaitGroup{} //等待所有goroutine
var waitAllForLog sync.WaitGroup
var waitAllForRedis sync.WaitGroup

var stopForLog int32 //
var stop int32       //停止标志

var gocount int32 //goroutine数量
var goid uint32
var DefLog *Log //日志

var msgQueId uint32 //消息队列id

var msgqueMapSync sync.Mutex
var goId uint64
var msgqueMap = map[uint32]IMsgQue{}

var stopMap = map[uint64]chan struct{}{}
var stopMapLock sync.Mutex
var stopMapForLog = map[uint64]chan struct{}{}
var stopMapForLogLock sync.Mutex

var stopChan chan os.Signal
var StartTick int64 = 0
var NowTick int64 = 0
var Timestamp int64 = 0
var TimeNanoStamp int64 = 0
var UdpServerGoCnt int = 32

var stopCheckIndex uint64 = 0

var stopCheckMap = struct {
	sync.Mutex
	M  map[uint64]string
	IM map[uint64]int64
}{M: map[uint64]string{}, IM: map[uint64]int64{}}

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	DefLog = NewLog(10000)
	DefLog.SetLogger(&ConsoleLogger{true}, true)
	timerTick()
}
