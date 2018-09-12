package antnet

import (
	"sync/atomic"
)

func Go(fn func()) {
	pc := Config.PoolSize + 1
	select {
	case poolChan <- fn:
		return
	default:
		pc = atomic.AddInt32(&poolGoCount, 1)
		if pc > Config.PoolSize {
			atomic.AddInt32(&poolGoCount, -1)
		}
	}

	waitAll.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		debugStr = LogSimpleStack()
		LogDebug("goroutine start id:%d count:%d from:%s", id, c, debugStr)
	}
	go func() {
		Try(fn, nil)
		for pc <= Config.PoolSize {
			select {
			case <-stopChanForGo:
				pc = Config.PoolSize + 1
			case nfn := <-poolChan:
				Try(nfn, nil)
			}
		}

		waitAll.Done()
		c = atomic.AddInt32(&gocount, -1)

		if DefLog.Level() <= LogLevelDebug {
			LogDebug("goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func Go2(fn func(cstop chan struct{})) {
	waitAll.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		debugStr = LogSimpleStack()
		LogDebug("goroutine start id:%d count:%d from:%s", id, c, debugStr)
	}

	go func() {
		Try(func() { fn(stopChanForGo) }, nil)
		waitAll.Done()
		c = atomic.AddInt32(&gocount, -1)
		if DefLog.Level() <= LogLevelDebug {
			LogDebug("goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func GoArgs(fn func(...interface{}), args ...interface{}) {
	waitAll.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		debugStr = LogSimpleStack()
		LogDebug("goroutine start id:%d count:%d from:%s", id, c, debugStr)
	}

	go func() {
		Try(func() { fn(args...) }, nil)

		waitAll.Done()
		c = atomic.AddInt32(&gocount, -1)
		if DefLog.Level() <= LogLevelDebug {
			LogDebug("goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func goForRedis(fn func()) {
	waitAllForRedis.Add(1)
	var debugStr string
	id := atomic.AddUint32(&goid, 1)
	c := atomic.AddInt32(&gocount, 1)
	if DefLog.Level() <= LogLevelDebug {
		debugStr = LogSimpleStack()
		LogDebug("goroutine start id:%d count:%d from:%s", id, c, debugStr)
	}
	go func() {
		Try(fn, nil)
		waitAllForRedis.Done()
		c = atomic.AddInt32(&gocount, -1)

		if DefLog.Level() <= LogLevelDebug {
			LogDebug("goroutine end id:%d count:%d from:%s", id, c, debugStr)
		}
	}()
}

func goForLog(fn func(cstop chan struct{})) bool {
	if IsStop() {
		return false
	}
	waitAllForLog.Add(1)

	go func() {
		fn(stopChanForLog)
		waitAllForLog.Done()
	}()
	return true
}
