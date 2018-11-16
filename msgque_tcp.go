package antnet

import (
	"bufio"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type tcpMsgQue struct {
	msgQue
	conn       net.Conn     //连接
	listener   net.Listener //监听
	network    string
	address    string
	wait       sync.WaitGroup
	connecting int32
	rawBuffer  []byte
}

func (r *tcpMsgQue) SetCmdReadRaw() {
	r.rawBuffer = make([]byte, Config.ReadDataBuffer)
}

func (r *tcpMsgQue) GetNetType() NetType {
	return NetTypeTcp
}
func (r *tcpMsgQue) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		Go(func() {
			if r.init {
				r.handler.OnDelMsgQue(r)
				if r.connecting == 1 {
					r.available = false
					return
				}
			}
			r.available = false
			r.baseStop()
		})
	}
}

func (r *tcpMsgQue) IsStop() bool {
	if r.stop == 0 {
		if IsStop() {
			r.Stop()
		}
	}
	return r.stop == 1
}

func (r *tcpMsgQue) LocalAddr() string {
	if r.conn != nil {
		return r.conn.LocalAddr().String()
	} else if r.listener != nil {
		return r.listener.Addr().String()
	}
	return ""
}

func (r *tcpMsgQue) RemoteAddr() string {
	if r.realRemoteAddr != "" {
		return r.realRemoteAddr
	}
	if r.conn != nil {
		return r.conn.RemoteAddr().String()
	}
	return r.address
}

func (r *tcpMsgQue) readMsg() {
	headData := make([]byte, MsgHeadSize)
	var data []byte
	var head *MessageHead

	for !r.IsStop() {
		if head == nil {
			_, err := io.ReadFull(r.conn, headData)
			if err != nil {
				if err != io.EOF {
					LogDebug("msgque:%v recv data err:%v", r.id, err)
				}
				break
			}
			if head = NewMessageHead(headData); head == nil {
				LogError("msgque:%v read msg head failed", r.id)
				break
			}
			if head.Len == 0 {
				if !r.processMsg(r, &Message{Head: head}) {
					LogError("msgque:%v process msg cmd:%v act:%v", r.id, head.Cmd, head.Act)
					break
				}
				head = nil
			} else {
				data = make([]byte, head.Len)
			}
		} else {
			_, err := io.ReadFull(r.conn, data)
			if err != nil {
				LogError("msgque:%v recv data err:%v", r.id, err)
				break
			}

			if !r.processMsg(r, &Message{Head: head, Data: data}) {
				LogError("msgque:%v process msg cmd:%v act:%v", r.id, head.Cmd, head.Act)
				break
			}

			head = nil
			data = nil
		}
		r.lastTick = Timestamp
	}
}

func (r *tcpMsgQue) writeMsgFast() {
	var m *Message
	var data []byte
	gm := r.getGMsg(false)
	writeCount := 0
	tick := time.NewTimer(time.Second * time.Duration(r.timeout))
	for !r.IsStop() || m != nil {
		if m == nil {
			select {
			case <-stopChanForGo:
			case m = <-r.cwrite:
				if m != nil {
					data = m.Bytes()
				}
			case <-gm.c:
				if gm.fun == nil || gm.fun(r) {
					m = gm.msg
					data = m.Bytes()
				}
				gm = r.getGMsg(true)
			case <-tick.C:
				if r.isTimeout(tick) {
					r.Stop()
				}
			}
		}

		if m == nil {
			continue
		}

		if writeCount < len(data) {
			n, err := r.conn.Write(data[writeCount:])
			if err != nil {
				LogError("msgque write id:%v err:%v", r.id, err)
				break
			}
			writeCount += n
		}

		if writeCount == len(data) {
			writeCount = 0
			m = nil
		}
		r.lastTick = Timestamp
	}
	tick.Stop()
}

func (r *tcpMsgQue) writeMsg() {
	var m *Message
	head := make([]byte, MsgHeadSize)
	gm := r.getGMsg(false)
	writeCount := 0
	tick := time.NewTimer(time.Second * time.Duration(r.timeout))
	for !r.IsStop() || m != nil {
		if m == nil {
			select {
			case <-stopChanForGo:
			case m = <-r.cwrite:
				if m != nil {
					m.Head.FastBytes(head)
				}
			case <-gm.c:
				if gm.fun == nil || gm.fun(r) {
					m = gm.msg
					m.Head.FastBytes(head)
				}
				gm = r.getGMsg(true)
			case <-tick.C:
				if r.isTimeout(tick) {
					r.Stop()
				}
			}
		}

		if m == nil {
			continue
		}

		if writeCount < MsgHeadSize {
			n, err := r.conn.Write(head[writeCount:])
			if err != nil {
				LogError("msgque write id:%v err:%v", r.id, err)
				break
			}
			writeCount += n
		}

		if writeCount >= MsgHeadSize && m.Data != nil {
			n, err := r.conn.Write(m.Data[writeCount-MsgHeadSize : int(m.Head.Len)])
			if err != nil {
				LogError("msgque write id:%v err:%v", r.id, err)
				break
			}
			writeCount += n
		}

		if writeCount == int(m.Head.Len)+MsgHeadSize {
			writeCount = 0
			m = nil
		}
		r.lastTick = Timestamp
	}
	tick.Stop()
}

func (r *tcpMsgQue) readCmd() {
	reader := bufio.NewReader(r.conn)
	var err error
	var len int
	var data []byte
	for !r.IsStop() {
		if r.rawBuffer != nil {
			len, err = reader.Read(r.rawBuffer)
			if err == nil && len > 0 {
				data = make([]byte, len)
				copy(data, r.rawBuffer)
			}
		} else {
			data, err = reader.ReadBytes('\n')
		}
		if err != nil {
			break
		}
		if !r.processMsg(r, &Message{Data: data}) {
			break
		}
		r.lastTick = Timestamp
	}
}

func (r *tcpMsgQue) writeCmd() {
	var m *Message
	gm := r.getGMsg(false)
	writeCount := 0
	tick := time.NewTimer(time.Second * time.Duration(r.timeout))
	for !r.IsStop() || m != nil {
		if m == nil {
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
		}

		if m == nil || m.Data == nil {
			m = nil
			continue
		}
		n, err := r.conn.Write(m.Data[writeCount:])
		if err != nil {
			LogError("msgque write id:%v err:%v", r.id, err)
			break
		}
		writeCount += n
		if writeCount == len(m.Data) {
			writeCount = 0
			m = nil
		}
		r.lastTick = Timestamp
	}
	tick.Stop()
}

func (r *tcpMsgQue) read() {
	defer func() {
		r.wait.Done()
		if err := recover(); err != nil {
			LogError("msgque read panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		r.Stop()
	}()

	r.wait.Add(1)
	if r.msgTyp == MsgTypeCmd {
		r.readCmd()
	} else {
		r.readMsg()
	}
}

func (r *tcpMsgQue) write() {
	defer func() {
		r.wait.Done()
		if err := recover(); err != nil {
			LogError("msgque write panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		if r.conn != nil {
			r.conn.Close()
		}
		r.Stop()
	}()
	r.wait.Add(1)
	if r.msgTyp == MsgTypeCmd {
		r.writeCmd()
	} else {
		if r.sendFast {
			r.writeMsgFast()
		} else {
			r.writeMsg()
		}
	}
}

func (r *tcpMsgQue) listen() {
	c := make(chan struct{})
	Go2(func(cstop chan struct{}) {
		select {
		case <-cstop:
		case <-c:
		}
		r.listener.Close()
	})
	for !r.IsStop() {
		c, err := r.listener.Accept()
		if err != nil {
			if stop == 0 && r.stop == 0 {
				LogError("accept failed msgque:%v err:%v", r.id, err)
			}
			break
		} else {
			Go(func() {
				msgque := newTcpAccept(c, r.msgTyp, r.handler, r.parserFactory)
				if r.handler.OnNewMsgQue(msgque) {
					msgque.init = true
					msgque.available = true
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
				} else {
					msgque.Stop()
				}
			})
		}
	}

	close(c)
	r.Stop()
}

func (r *tcpMsgQue) connect() {
	LogInfo("connect to addr:%s msgque:%d", r.address, r.id)
	c, err := net.DialTimeout(r.network, r.address, time.Second)
	if err != nil {
		LogInfo("connect to addr:%s failed msgque:%d", r.address, r.id)
		r.handler.OnConnectComplete(r, false)
		atomic.CompareAndSwapInt32(&r.connecting, 1, 0)
		r.Stop()
	} else {
		r.conn = c
		r.available = true
		LogInfo("connect to addr:%s ok msgque:%d", r.address, r.id)
		if r.handler.OnConnectComplete(r, true) {
			atomic.CompareAndSwapInt32(&r.connecting, 1, 0)
			Go(func() {
				LogInfo("process read for msgque:%d", r.id)
				r.read()
				LogInfo("process read end for msgque:%d", r.id)
			})
			Go(func() {
				LogInfo("process write for msgque:%d", r.id)
				r.write()
				LogInfo("process write end for msgque:%d", r.id)
			})
		} else {
			atomic.CompareAndSwapInt32(&r.connecting, 1, 0)
			r.Stop()
		}
	}
}

func (r *tcpMsgQue) Reconnect(t int) {
	if IsStop() {
		return
	}
	if r.conn != nil {
		if r.stop == 0 {
			return
		}
	}

	if !atomic.CompareAndSwapInt32(&r.connecting, 0, 1) {
		return
	}

	if r.init {
		if t < 1 {
			t = 1
		}
	}
	r.init = true
	Go(func() {
		if len(r.cwrite) == 0 {
			r.cwrite <- nil
		}
		r.wait.Wait()
		if t > 0 {
			SetTimeout(t*1000, func(arg ...interface{}) int {
				r.stop = 0
				r.connect()
				return 0
			})
		} else {
			r.stop = 0
			r.connect()
		}

	})
}

func newTcpConn(network, addr string, conn net.Conn, msgtyp MsgType, handler IMsgHandler, parser IParserFactory, user interface{}) *tcpMsgQue {
	msgque := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgqueId, 1),
			cwrite:        make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeConn,
			gmsgId:        gmsgId,
			parserFactory: parser,
			lastTick:      Timestamp,
			user:          user,
		},
		conn:    conn,
		network: network,
		address: addr,
	}
	if parser != nil {
		msgque.parser = parser.Get()
	}
	msgqueMapSync.Lock()
	msgqueMap[msgque.id] = &msgque
	msgqueMapSync.Unlock()
	LogInfo("new msgque id:%d connect to addr:%s:%s", msgque.id, network, addr)
	return &msgque
}

func newTcpAccept(conn net.Conn, msgtyp MsgType, handler IMsgHandler, parser IParserFactory) *tcpMsgQue {
	msgque := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgqueId, 1),
			cwrite:        make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeAccept,
			gmsgId:        gmsgId,
			lastTick:      Timestamp,
			parserFactory: parser,
		},
		conn: conn,
	}
	if parser != nil {
		msgque.parser = parser.Get()
	}
	msgqueMapSync.Lock()
	msgqueMap[msgque.id] = &msgque
	msgqueMapSync.Unlock()
	LogInfo("new msgque id:%d from addr:%s", msgque.id, conn.RemoteAddr().String())
	return &msgque
}

func newTcpListen(listener net.Listener, msgtyp MsgType, handler IMsgHandler, parser IParserFactory, addr string) *tcpMsgQue {
	msgque := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgqueId, 1),
			msgTyp:        msgtyp,
			handler:       handler,
			parserFactory: parser,
			connTyp:       ConnTypeListen,
		},
		listener: listener,
	}

	msgqueMapSync.Lock()
	msgqueMap[msgque.id] = &msgque
	msgqueMapSync.Unlock()
	LogInfo("new tcp listen id:%d addr:%s", msgque.id, addr)
	return &msgque
}
