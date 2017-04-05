package antnet

import (
	"net"
	"runtime"
	"sync"
	"sync/atomic"
)

type udpMsgQue struct {
	msgQue
	conn     *net.UDPConn //连接
	cread    chan []byte  //写入通道
	addr     *net.UDPAddr
	lastTick int64
}

func (r *udpMsgQue) GetNetType() NetType {
	return NetTypeUdp
}
func (r *udpMsgQue) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		if r.init {
			r.handler.OnDelMsgQue(r)
		}

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
			buf := make([]byte, 1<<12)
			LogError(string(buf[:runtime.Stack(buf, false)]))
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
		var head *MessageHead
		if r.msgTyp == MsgTypeCmd {
			msg = &Message{Data: data}
		} else {
			if head = NewMessageHead(data); head == nil {
				break
			}
			if head.Len > 0 {
				msg = &Message{Head: head, Data: data[MsgHeadSize:]}
			} else {
				msg = &Message{Head: head}
			}
		}

		if !r.init {
			if !r.handler.OnNewMsgQue(r) {
				break
			}
			SetTimeout(r.timeout*1000, func(args ...interface{}) uint32 {
				left := int(NowTick - r.lastTick)
				if left >= r.timeout*1000 {
					r.Stop()
					return 0
				} else {
					return uint32(r.timeout - left)
				}
			}, nil)
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
			r.Stop()
		}
	}()
	var m *Message
	for !r.IsStop() {
		select {
		case m = <-r.cwrite:
		}
		if m == nil {
			break
		}

		if r.msgTyp == MsgTypeCmd {
			if m.Data != nil {
				r.conn.WriteToUDP(m.Data, r.addr)
			}
		} else {
			if m.Head != nil && m.Data != nil {
				data := make([]byte, m.Head.Len+MsgHeadSize)
				copy(data, m.Head.Bytes())
				copy(data[MsgHeadSize:], m.Data)
				r.conn.WriteToUDP(data, r.addr)
			} else {
				r.conn.WriteToUDP(m.Head.Bytes(), r.addr)
			}
		}
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

func (r *udpMsgQue) listen() {
	data := make([]byte, 1<<22)
	for !r.IsStop() {
		n, addr, err := r.conn.ReadFromUDP(data)
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

	r.Stop()
	r.conn.Close()
}

func newUdpAccept(conn *net.UDPConn, msgtyp MsgType, handler IMsgHandler, parser *Parser, addr *net.UDPAddr) *udpMsgQue {
	msgque := udpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueId, 1),
			cwrite:        make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeAccept,
			parserFactory: parser,
		},
		conn:     conn,
		cread:    make(chan []byte, 64),
		addr:     addr,
		lastTick: NowTick,
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
			parserFactory: parser,
			connTyp:       ConnTypeListen,
		},
		conn: conn,
	}

	msgqueMapSync.Lock()
	msgqueMap[msgque.id] = &msgque
	msgqueMapSync.Unlock()
	LogInfo("new udp listen id:%d addr:%s", msgque.id, addr)
	return &msgque
}
