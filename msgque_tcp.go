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
	available  bool
	wait       sync.WaitGroup
	connecting int32
}

func (r *tcpMsgQue) GetNetType() NetType {
	return NetTypeTcp
}
func (r *tcpMsgQue) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		r.available = false
		if r.init {
			r.handler.OnDelMsgQue(r)
			if r.connecting == 1 {
				return
			}
		}

		if r.listener != nil {
			if tcp, ok := r.listener.(*net.TCPListener); ok {
				tcp.Close()
			}
		}

		r.BaseStop()
	}
}

func (r *tcpMsgQue) Available() bool {
	return r.available
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
	if r.conn != nil {
		return r.conn.RemoteAddr().String()
	}
	return ""
}

func (r *tcpMsgQue) readMsg() {
	headData := make([]byte, MsgHeadSize)
	var data []byte
	var head *MessageHead

	for !r.IsStop() {
		if r.timeout > 0 {
			r.conn.SetReadDeadline(time.Now().Add(time.Duration(r.timeout) * time.Second))
		}
		if head == nil {
			_, err := io.ReadFull(r.conn, headData)
			if err != nil {
				if err != io.EOF {
					LogError("msgque:%v recv data err:%v", r.id, err)
				}
				break
			}
			if head = NewMessageHead(headData); head == nil {
				LogError("msgque:%v read msg head err:%v", r.id)
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
	}
}

func (r *tcpMsgQue) writeMsg() {
	var m *Message
	var head []byte
	writeCount := 0
	for !r.IsStop() || m != nil {
		if m == nil {
			select {
			case m = <-r.cwrite:
				if m != nil {
					head = m.Head.Bytes()
				}
			}
		}
		if m != nil {
			if r.timeout > 0 {
				r.conn.SetWriteDeadline(time.Now().Add(time.Duration(r.timeout) * time.Second))
			}
			if writeCount < MsgHeadSize {
				n, err := r.conn.Write(head[writeCount:])
				if err != nil {
					LogError("msgque write id:%v err:%v", r.id, err)
					r.Stop()
					break
				}
				writeCount += n
			}

			if writeCount >= MsgHeadSize && m.Data != nil {
				n, err := r.conn.Write(m.Data[writeCount-MsgHeadSize : int(m.Head.Len)])
				if err == io.EOF {
					LogError("msgque write id:%v err:%v", r.id, err)
					break
				}
				writeCount += n
			}

			if writeCount == int(m.Head.Len)+MsgHeadSize {
				writeCount = 0
				m = nil
			}
		}
	}
}

func (r *tcpMsgQue) readCmd() {
	reader := bufio.NewReader(r.conn)
	for !r.IsStop() {
		if r.timeout > 0 {
			r.conn.SetReadDeadline(time.Now().Add(time.Duration(r.timeout) * time.Second))
		}
		data, err := reader.ReadBytes('\n')
		if err != nil {
			break
		}
		if !r.processMsg(r, &Message{Data: data}) {
			break
		}
	}
}

func (r *tcpMsgQue) writeCmd() {
	var m *Message
	writeCount := 0
	for !r.IsStop() || m != nil {
		if m == nil {
			select {
			case m = <-r.cwrite:
			}
		}
		if m != nil {
			if r.timeout > 0 {
				r.conn.SetWriteDeadline(time.Now().Add(time.Duration(r.timeout) * time.Second))
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
		}
	}
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
		r.writeMsg()
	}
}

func (r *tcpMsgQue) listen() {
	for !r.IsStop() {
		c, err := r.listener.Accept()
		if err != nil {
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
		LogInfo("connect to addr:%s ok msgque:%d", r.address, r.id)
		if r.handler.OnConnectComplete(r, true) {
			atomic.CompareAndSwapInt32(&r.connecting, 1, 0)
			r.available = true
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
		if r.conn != nil {
			r.conn.Close()
			if len(r.cwrite) == 0 {
				r.cwrite <- nil
			}
			r.wait.Wait()
		}
		r.stop = 0
		if t > 0 {
			SetTimeout(t*1000, func(arg ...interface{}) int {
				r.connect()
				return 0
			})
		} else {
			r.connect()
		}

	})
}

func newTcpConn(network, addr string, conn net.Conn, msgtyp MsgType, handler IMsgHandler, parser *Parser, user interface{}) *tcpMsgQue {
	msgque := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueId, 1),
			cwrite:        make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeConn,
			parserFactory: parser,
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

func newTcpAccept(conn net.Conn, msgtyp MsgType, handler IMsgHandler, parser *Parser) *tcpMsgQue {
	msgque := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueId, 1),
			cwrite:        make(chan *Message, 64),
			msgTyp:        msgtyp,
			handler:       handler,
			timeout:       DefMsgQueTimeout,
			connTyp:       ConnTypeAccept,
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

func newTcpListen(listener net.Listener, msgtyp MsgType, handler IMsgHandler, parser *Parser, addr string) *tcpMsgQue {
	msgque := tcpMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgQueId, 1),
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
