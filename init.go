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
	PoolGoCount int32
}

var statis = &Statis{}

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

var waitAll = &WaitGroup{} //等待所有goroutine
var waitAllForLog sync.WaitGroup
var waitAllForRedis sync.WaitGroup

var stopForLog int32 //
var stop int32       //停止标志

var gocount int32 //goroutine数量
var goid uint32
var DefLog *Log //日志

var msgqueId uint32 //消息队列id
var msgqueMapSync sync.Mutex
var msgqueMap = map[uint32]IMsgQue{}

type gMsg struct {
	c   chan struct{}
	msg *Message
	fun func(msgque IMsgQue) bool
}

var gmsgId uint16
var gmsgMapSync sync.Mutex
var gmsgArray = [65536]*gMsg{}

var atexitId uint32
var atexitMapSync sync.Mutex
var atexitMap = map[uint32]func(){}

var stopChanForGo = make(chan struct{})
var stopChanForLog = make(chan struct{})
var stopChanForSys = make(chan os.Signal, 1)

var poolChan = make(chan func())
var poolGoCount int32

var StartTick int64
var NowTick int64
var Timestamp int64

var Config = struct {
	AutoCompressLen  uint32
	UdpServerGoCnt   int
	TimeSyncInterval uint
	PoolSize         int32
	SSLCrtPath       string
	SSLKeyPath       string
	EnableWss        bool
	ReadDataBuffer   int
	StopTimeout      int
}{UdpServerGoCnt: 64, TimeSyncInterval: 10, PoolSize: 50000, ReadDataBuffer: 1 << 12, StopTimeout:3000}

var stopCheckIndex uint64
var stopCheckMap = struct {
	sync.Mutex
	M map[uint64]string
}{M: map[uint64]string{}}

func init() {
	gmsgArray[gmsgId] = &gMsg{c: make(chan struct{})}
	runtime.GOMAXPROCS(runtime.NumCPU())
	DefLog = NewLog(10000, &ConsoleLogger{true})
	DefLog.SetLevel(LogLevelInfo)
	timerTick()
}
