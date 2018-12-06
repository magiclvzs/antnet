package antnet

import (
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type wsMsgQue struct {
	msgQue
	conn     *websocket.Conn
	upgrader *websocket.Upgrader
	addr     string
	url      string
	listener *http.Server
}

func (r *wsMsgQue) GetNetType() NetType {
	return NetTypeWs
}

func (r *wsMsgQue) Stop() {
	if atomic.CompareAndSwapInt32(&r.stop, 0, 1) {
		Go(func() {
			if r.init {
				r.handler.OnDelMsgQue(r)
			}
			r.available = false
			r.baseStop()
		})
	}
}

func (r *wsMsgQue) IsStop() bool {
	if r.stop == 0 {
		if IsStop() {
			r.Stop()
		}
	}
	return r.stop == 1
}

func (r *wsMsgQue) LocalAddr() string {
	if r.conn != nil {
		return r.conn.LocalAddr().String()
	}
	return ""
}

func (r *wsMsgQue) RemoteAddr() string {
	if r.realRemoteAddr != "" {
		return r.realRemoteAddr
	}
	if r.conn != nil {
		return r.conn.RemoteAddr().String()
	}
	return ""
}

func (r *wsMsgQue) readCmd() {
	for !r.IsStop() {
		_, data, err := r.conn.ReadMessage()
		if err != nil {
			break
		}
		if !r.processMsg(r, &Message{Data: data}) {
			break
		}
		r.lastTick = Timestamp
	}
}

func (r *wsMsgQue) writeCmd() {
	var m *Message
	gm := r.getGMsg(false)
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
		err := r.conn.WriteMessage(websocket.BinaryMessage, m.Data)
		if err != nil {
			LogError("msgque write id:%v err:%v", r.id, err)
			break
		}
		m = nil
		r.lastTick = Timestamp
	}
	tick.Stop()
}

func (r *wsMsgQue) read() {
	defer func() {
		if err := recover(); err != nil {
			LogError("msgque read panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		r.Stop()
	}()

	r.readCmd()
}

func (r *wsMsgQue) write() {
	defer func() {
		if err := recover(); err != nil {
			LogError("msgque write panic id:%v err:%v", r.id, err.(error))
			LogStack()
		}
		if r.conn != nil {
			r.conn.Close()
		}
		r.Stop()
	}()

	r.writeCmd()
}

func (r *wsMsgQue) listen() {
	Go2(func(cstop chan struct{}) {
		select {
		case <-cstop:
		}
		r.listener.Close()
	})

	r.upgrader = &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	http.HandleFunc(r.url, func(hw http.ResponseWriter, hr *http.Request) {
		c, err := r.upgrader.Upgrade(hw, hr, nil)
		if err != nil {
			if stop == 0 && r.stop == 0 {
				LogError("accept failed msgque:%v err:%v", r.id, err)
			}
		} else {
			Go(func() {
				msgque := newWsAccept(c, r.msgTyp, r.handler, r.parserFactory)
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
	})

	if Config.EnableWss {
		if Config.SSLCrtPath != "" && Config.SSLKeyPath != "" {
			r.listener.ListenAndServeTLS(Config.SSLCrtPath, Config.SSLKeyPath)
		} else {
			LogError("start wss failed ssl path not set please set now auto change to ws")
			r.listener.ListenAndServe()
		}
	} else {
		r.listener.ListenAndServe()
	}
}

func newWsAccept(conn *websocket.Conn, msgtyp MsgType, handler IMsgHandler, parser IParserFactory) *wsMsgQue {
	msgque := wsMsgQue{
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

func newWsListen(addr, url string, msgtyp MsgType, handler IMsgHandler, parser IParserFactory) *wsMsgQue {
	msgque := wsMsgQue{
		msgQue: msgQue{
			id:            atomic.AddUint32(&msgqueId, 1),
			msgTyp:        msgtyp,
			handler:       handler,
			parserFactory: parser,
			connTyp:       ConnTypeListen,
		},
		addr:     addr,
		url:      url,
		listener: &http.Server{Addr: addr},
	}

	msgqueMapSync.Lock()
	msgqueMap[msgque.id] = &msgque
	msgqueMapSync.Unlock()
	LogInfo("new ws listen id:%d addr:%s url:%s", msgque.id, addr, url)
	return &msgque
}
