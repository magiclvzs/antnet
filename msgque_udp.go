package antnet

import (
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type udpMsgQue struct {
	msgQue
	conn  *net.UDPConn //连接
	cread chan []byte  //写入通道
	addr  *net.UDPAddr
	sync.Mutex
}

type udpMsgQueHelper struct {
	init int32
	null bool
	sync.Mutex
	*udpMsgQue
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
			r.baseStop()
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
	if r.realRemoteAddr != "" {
		return r.realRemoteAddr
	}
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
	gm := r.getGMsg(false)
	tick := time.NewTimer(time.Second * time.Duration(r.timeout))
	for !r.IsStop() {
		var m *Message = nil
		select {
		case <-stopChanForGo:
		case m = <-r.cwrite:
		case <-gm.c:
			if gm.fun == nil || gm.fun(r) {
				m = gm.msg
			}
			gm = r.getGMsg(true)
		case <-tick.C:
			if r.isTimeout(tick) {
				r.Stop()
			}
		}

		if m == nil {
			continue
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

	tick.Stop()
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

var udpMap map[string]*udpMsgQueHelper = map[string]*udpMsgQueHelper{}
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

		addrStr := addr.String()
		udpMapLock.Lock()
		helper, ok := udpMap[addrStr]
		if !ok {
			helper = &udpMsgQueHelper{null: true}
			udpMap[addrStr] = helper
		}
		udpMapLock.Unlock()

		if helper.null {
			helper.Lock()
			if atomic.CompareAndSwapInt32(&helper.init, 0, 1) {
				helper.udpMsgQue = newUdpAccept(r.conn, r.msgTyp, r.handler, r.parserFactory, addr)
				helper.null = false
			}
			helper.Unlock()
		}

		if !helper.sendRead(data, n) {
			LogError("drop msg because msgque full msgqueid:%v", helper.id)
		}
	}
}

func (r *udpMsgQue) listen() {
	for i := 0; i < Config.UdpServerGoCnt; i++ {
		Go(func() {
			r.listenTrue()
		})
	}
	c := make(chan struct{})
	Go2(func(cstop chan struct{}) {
		select {
		case <-cstop:
		case <-c:
		}
		r.conn.Close()
	})
	r.listenTrue()
	close(c)
	r.Stop()
}

func newUdpAccept(conn *net.UDPConn, msgtyp MsgType, handler IMsgHandler, parser IParserFactory, addr *net.UDPAddr) *udpMsgQue {
	msgque := udpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgqueId, 1),
			cwrite:        make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			available:     true,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeAccept,
			gmsgId:        gmsgId,
			parserFactory: parser,
			lastTick:      Timestamp,
		},
		conn:  conn,
		cread: make(chan []byte, 64),
		addr:  addr,
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

func newUdpListen(conn *net.UDPConn, msgtyp MsgType, handler IMsgHandler, parser IParserFactory, addr string) *udpMsgQue {
	msgque := udpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgqueId, 1),
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
