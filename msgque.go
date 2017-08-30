package antnet

import (
	"net"
	"reflect"
	"strings"
	"sync"
)

var DefMsgQueTimeout int = 180

type MsgType int

const (
	MsgTypeMsg MsgType = iota //消息基于确定的消息头
	MsgTypeCmd                //消息没有消息头，以\n分割
)

type NetType int

const (
	NetTypeTcp NetType = iota //TCP类型
	NetTypeUdp                //UDP类型
)

type ConnType int

const (
	ConnTypeListen ConnType = iota //监听
	ConnTypeConn                   //连接产生的
	ConnTypeAccept                 //Accept产生的
)

type IMsgQue interface {
	Id() uint32
	GetMsgType() MsgType
	GetConnType() ConnType
	GetNetType() NetType

	LocalAddr() string
	RemoteAddr() string

	Stop()
	IsStop() bool
	Available() bool

	Send(m *Message) (re bool)
	SendString(str string) (re bool)
	SendStringLn(str string) (re bool)
	SendByteStr(str []byte) (re bool)
	SendByteStrLn(str []byte) (re bool)
	SendCallback(m *Message, c chan *Message) (re bool)
	SetTimeout(t int)
	GetTimeout() int
	Reconnect(t int) //重连间隔  最小1s，此函数仅能连接关闭是调用

	GetHandler() IMsgHandler

	SetUser(user interface{})
	GetUser() interface{}

	tryCallback(msg *Message) (re bool)
}

type msgQue struct {
	id uint32 //唯一标示

	cwrite  chan *Message //写入通道
	stop    int32         //停止标记
	msgTyp  MsgType       //消息类型
	connTyp ConnType      //通道类型

	handler       IMsgHandler //处理者
	parser        IParser
	parserFactory *Parser
	timeout       int //传输超时

	init         bool
	callback     map[int]chan *Message
	user         interface{}
	callbackLock sync.Mutex
}

func (r *msgQue) SetUser(user interface{}) {
	r.user = user
}

func (r *msgQue) GetUser() interface{} {
	return r.user
}

func (r *msgQue) GetHandler() IMsgHandler {
	return r.handler
}
func (r *msgQue) Available() bool {
	return true
}
func (r *msgQue) GetMsgType() MsgType {
	return r.msgTyp
}

func (r *msgQue) GetConnType() ConnType {
	return r.connTyp
}

func (r *msgQue) Id() uint32 {
	return r.id
}

func (r *msgQue) SetTimeout(t int) {
	r.timeout = t
}

func (r *msgQue) GetTimeout() int {
	return r.timeout
}
func (r *msgQue) Reconnect(t int) {

}

func (r *msgQue) Send(m *Message) (re bool) {
	if m == nil {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			re = false
		}
	}()

	r.cwrite <- m
	return true
}

func (r *msgQue) SendCallback(m *Message, c chan *Message) (re bool) {
	if c == nil || cap(c) < 1 {
		LogError("try send callback but chan is null or no buffer")
		return
	}
	if r.Send(m) {
		r.setCallback(m.Tag(), c)
	} else {
		c <- nil
		return
	}
	return true
}

func (r *msgQue) SendString(str string) (re bool) {
	return r.Send(&Message{Data: []byte(str)})
}

func (r *msgQue) SendStringLn(str string) (re bool) {
	return r.SendString(str + "\n")
}

func (r *msgQue) SendByteStr(str []byte) (re bool) {
	return r.SendString(string(str))
}

func (r *msgQue) SendByteStrLn(str []byte) (re bool) {
	return r.SendString(string(str) + "\n")
}

func (r *msgQue) tryCallback(msg *Message) (re bool) {
	if r.callback == nil {
		return false
	}
	defer func() {
		if err := recover(); err != nil {

		}
		r.callbackLock.Unlock()
	}()
	r.callbackLock.Lock()
	if r.callback != nil {
		tag := msg.Tag()
		if c, ok := r.callback[tag]; ok {
			delete(r.callback, tag)
			c <- msg
			re = true
		}
	}
	return
}

func (r *msgQue) setCallback(tag int, c chan *Message) {
	defer func() {
		if err := recover(); err != nil {

		}
		r.callback[tag] = c
		r.callbackLock.Unlock()
	}()

	r.callbackLock.Lock()
	if r.callback == nil {
		r.callback = make(map[int]chan *Message)
	}
	oc, ok := r.callback[tag]
	if ok { //可能已经关闭
		oc <- nil
	}
}

func (r *msgQue) BaseStop() {
	if r.cwrite != nil {
		close(r.cwrite)
	}

	for k, v := range r.callback {
		v <- nil
		delete(r.callback, k)
	}
	msgqueMapSync.Lock()
	delete(msgqueMap, r.id)
	msgqueMapSync.Unlock()
	LogInfo("msgque close id:%d", r.id)
}

func (r *msgQue) processMsg(msgque IMsgQue, msg *Message) bool {
	if r.parser != nil && msg.Data != nil {
		mp, err := r.parser.ParseC2S(msg)
		if err == nil {
			msg.IMsgParser = mp
		} else {
			if r.parser.GetErrType() == ParseErrTypeSendRemind {
				if msg.Head != nil {
					r.Send(r.parser.GetRemindMsg(err, r.msgTyp).CopyTag(msg))
				} else {
					r.Send(r.parser.GetRemindMsg(err, r.msgTyp))
				}
				return true
			} else if r.parser.GetErrType() == ParseErrTypeClose {
				return false
			} else if r.parser.GetErrType() == ParseErrTypeContinue {
				return true
			}
		}
	}
	f := r.handler.GetHandlerFunc(msgque, msg)
	if f == nil {
		f = r.handler.OnProcessMsg
	}
	return f(msgque, msg)
}

type HandlerFunc func(msgque IMsgQue, msg *Message) bool

type IMsgHandler interface {
	OnNewMsgQue(msgque IMsgQue) bool                         //新的消息队列
	OnDelMsgQue(msgque IMsgQue)                              //消息队列关闭
	OnProcessMsg(msgque IMsgQue, msg *Message) bool          //默认的消息处理函数
	OnConnectComplete(msgque IMsgQue, ok bool) bool          //连接成功
	GetHandlerFunc(msgque IMsgQue, msg *Message) HandlerFunc //根据消息获得处理函数
}

type IMsgRegister interface {
	Register(cmd, act uint8, fun HandlerFunc)
	RegisterMsg(v interface{}, fun HandlerFunc)
}

type DefMsgHandler struct {
	msgMap  map[int]HandlerFunc
	typeMap map[reflect.Type]HandlerFunc
}

func (r *DefMsgHandler) OnNewMsgQue(msgque IMsgQue) bool                { return true }
func (r *DefMsgHandler) OnDelMsgQue(msgque IMsgQue)                     {}
func (r *DefMsgHandler) OnProcessMsg(msgque IMsgQue, msg *Message) bool { return true }
func (r *DefMsgHandler) OnConnectComplete(msgque IMsgQue, ok bool) bool { return true }
func (r *DefMsgHandler) GetHandlerFunc(msgque IMsgQue, msg *Message) HandlerFunc {
	if msgque.tryCallback(msg) {
		return r.OnProcessMsg
	}

	if msg.CmdAct() == 0 {
		if r.typeMap != nil {
			if f, ok := r.typeMap[reflect.TypeOf(msg.C2S())]; ok {
				return f
			}
		}
	} else if r.msgMap != nil {
		if f, ok := r.msgMap[msg.CmdAct()]; ok {
			return f
		}
	}

	return nil
}

func (r *DefMsgHandler) RegisterMsg(v interface{}, fun HandlerFunc) {
	msgType := reflect.TypeOf(v)
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		LogFatal("message pointer required")
		return
	}
	if r.typeMap == nil {
		r.typeMap = map[reflect.Type]HandlerFunc{}
	}
	r.typeMap[msgType] = fun
}

func (r *DefMsgHandler) Register(cmd, act uint8, fun HandlerFunc) {
	if r.msgMap == nil {
		r.msgMap = map[int]HandlerFunc{}
	}
	r.msgMap[CmdAct(cmd, act)] = fun
}

type EchoMsgHandler struct {
	DefMsgHandler
}

func (r *EchoMsgHandler) OnProcessMsg(msgque IMsgQue, msg *Message) bool {
	msgque.Send(msg)
	return true
}

func StartServer(addr string, typ MsgType, handler IMsgHandler, parser *Parser) error {
	addrs := strings.Split(addr, "://")
	if addrs[0] == "udp" || addrs[0] == "all" {
		naddr, err := net.ResolveUDPAddr("udp", addrs[1])
		if err != nil {
			LogError("listen on %s failed, errstr:%s", addr, err)
			return err
		}
		conn, err := net.ListenUDP("udp", naddr)
		if err != nil {
			LogError("listen on %s failed, errstr:%s", addr, err)
			return err
		}
		msgque := newUdpListen(conn, typ, handler, parser, addr)
		Go(func() {
			LogDebug("process listen for msgque:%d", msgque.id)
			msgque.listen()
			LogDebug("process listen end for msgque:%d", msgque.id)
		})
	}
	if addrs[0] == "tcp" || addrs[0] == "all" {
		listen, err := net.Listen("tcp", addrs[1])
		if err == nil {
			msgque := newTcpListen(listen, typ, handler, parser, addr)
			Go(func() {
				LogDebug("process listen for msgque:%d", msgque.id)
				msgque.listen()
				LogDebug("process listen end for msgque:%d", msgque.id)
			})
		} else {
			LogError("listen on %s failed, errstr:%s", addr, err)
		}
		return err
	}
	return nil
}

func StartConnect(netype string, addr string, typ MsgType, handler IMsgHandler, parser *Parser, user interface{}) IMsgQue {
	msgque := newTcpConn(netype, addr, nil, typ, handler, parser, user)
	if handler.OnNewMsgQue(msgque) {
		msgque.Reconnect(0)
		return msgque
	} else {
		msgque.Stop()
	}
	return nil
}
