package antnet

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type udpMsgQue struct {
	msgQue
	conn     *net.UDPConn //连接
	cread    chan []byte  //写入通道
	addr     *net.UDPAddr
	lastTick int64
	sync.Mutex
}

func (r *udpMsgQue) GetNetType() NetType {
	return NetTypeUdp
}

func (r *udpMsgQue) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		Go(func() {
			if r.init {
				r.handler.OnDelMsgQue(r)
			}
			r.available = false
			if r.cread != nil {
				close(r.cread)
			}

			udpMapLock.Lock()
			delete(udpMap, r.addr.String())
			udpMapLock.Unlock()

			if IsStop() && len(udpMap) == 0 && r.conn != nil {
				r.conn.Close()
			}
			r.BaseStop()
		})
	}
}

func (r *udpMsgQue) IsStop() bool {
	if r.stop == 0 {
		if IsStop() {
			r.Stop()
		}
	}
	return r.stop == 1
}

func (r *udpMsgQue) LocalAddr() string {
	if r.conn != nil {
		return r.conn.LocalAddr().String()
	}
	return ""
}

func (r *udpMsgQue) RemoteAddr() string {
	if r.addr != nil {
		return r.addr.String()
	}
	return ""
}

func (r *udpMsgQue) read() {
	defer func() {
		if err := recover(); err != nil {
			LogError("msgque read panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		r.Stop()
	}()
	var data []byte
	for !r.IsStop() {
		select {
		case data = <-r.cread:
		}
		if data == nil {
			break
		}
		var msg *Message
		if r.msgTyp == MsgTypeCmd {
			msg = &Message{Data: data}
		} else {
			head := MessageHeadFromByte(data)
			if head == nil {
				break
			}
			if head.Len > 0 {
				msg = &Message{Head: head, Data: data[MsgHeadSize:]}
			} else {
				msg = &Message{Head: head}
			}
		}
		r.lastTick = Timestamp
		if !r.init {
			if !r.handler.OnNewMsgQue(r) {
				break
			}
			r.init = true
		}

		if !r.processMsg(r, msg) {
			break
		}
	}
}

func (r *udpMsgQue) write() {
	defer func() {
		if err := recover(); err != nil {
			LogError("msgque write panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		r.Stop()
	}()

	timeouCheck := false
	tick := time.NewTimer(time.Second * time.Duration(r.timeout))
	for !r.IsStop() {
		var m *Message = nil
		select {
		case m = <-r.cwrite:
		case <-tick.C:
			left := int(Timestamp - r.lastTick)
			if left < r.timeout {
				timeouCheck = true
				tick = time.NewTimer(time.Second * time.Duration(r.timeout-left))
			}
		}
		if timeouCheck {
			timeouCheck = false
			continue
		}
		if m == nil {
			break
		}

		if r.msgTyp == MsgTypeCmd {
			if m.Data != nil {
				r.conn.WriteToUDP(m.Data, r.addr)
			}
		} else {
			if m.Head != nil || m.Data != nil {
				r.conn.WriteToUDP(m.Bytes(), r.addr)
			}
		}

		r.lastTick = Timestamp
	}
}

func (r *udpMsgQue) sendRead(data []byte, n int) (re bool) {
	defer func() {
		if err := recover(); err != nil {
			re = false
		}
	}()

	re = true
	if len(r.cread) < cap(r.cread) {
		pdata := make([]byte, n)
		copy(pdata, data)
		r.cread <- pdata
	}
	return
}

var udpMap map[string]*udpMsgQue = map[string]*udpMsgQue{}
var udpMapLock sync.Mutex

func (r *udpMsgQue) listenTrue() {
	data := make([]byte, 1<<16)
	for !r.IsStop() {
		r.Lock()
		n, addr, err := r.conn.ReadFromUDP(data)
		r.Unlock()
		if err != nil {
			if err.(net.Error).Timeout() {
				continue
			}
			break
		}

		if n <= 0 {
			continue
		}

		udpMapLock.Lock()
		msgque, ok := udpMap[addr.String()]
		if !ok {
			msgque = newUdpAccept(r.conn, r.msgTyp, r.handler, r.parserFactory, addr)
			udpMap[addr.String()] = msgque
		}
		udpMapLock.Unlock()

		if !msgque.sendRead(data, n) {
			LogError("drop msg because msgque full msgqueid:%v", msgque.id)
		}
	}
}

func (r *udpMsgQue) listen() {
	for i := 0; i < UdpServerGoCnt; i++ {
		Go(func() {
			r.listenTrue()
		})
	}
	r.listenTrue()
	r.Stop()
}

func newUdpAccept(conn *net.UDPConn, msgtyp MsgType, handler IMsgHandler, parser *Parser, addr *net.UDPAddr) *udpMsgQue {
	msgque := udpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueId, 1),
			cwrite:        make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			available:     true,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeAccept,
			parserFactory: parser,
		},
		conn:     conn,
		cread:    make(chan []byte, 64),
		addr:     addr,
		lastTick: Timestamp,
	}
	if parser != nil {
		msgque.parser = parser.Get()
	}
	msgqueMapSync.Lock()
	msgqueMap[msgque.id] = &msgque
	msgqueMapSync.Unlock()

	Go(func() {
		LogInfo("process read for msgque:%d", msgque.id)
		msgque.read()
		LogInfo("process read end for msgque:%d", msgque.id)
	})
	Go(func() {
		LogInfo("process write for msgque:%d", msgque.id)
		msgque.write()
		LogInfo("process write end for msgque:%d", msgque.id)
	})

	LogInfo("new msgque id:%d from addr:%s", msgque.id, addr.String())
	return &msgque
}

func newUdpListen(conn *net.UDPConn, msgtyp MsgType, handler IMsgHandler, parser *Parser, addr string) *udpMsgQue {
	msgque := udpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueId, 1),
			msgTyp:        msgtyp,
			handler:       handler,
			available:     true,
			parserFactory: parser,
			connTyp:       ConnTypeListen,
		},
		conn: conn,
	}
	conn.SetReadBuffer(1 << 24)
	conn.SetWriteBuffer(1 << 24)
	msgqueMapSync.Lock()
	msgqueMap[msgque.id] = &msgque
	msgqueMapSync.Unlock()
	LogInfo("new udp listen id:%d addr:%s", msgque.id, addr)
	return &msgque
}
