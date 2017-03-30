package antnet

import (
	"net"
	"reflect"
	"strings"
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

	Send(m *Message) (re bool)
	SendString(str string) (re bool)
	SendStringLn(str string) (re bool)
	SendByteStr(str []byte) (re bool)
	SendByteStrLn(str []byte) (re bool)
	SendCallback(m *Message, c chan interface{})
	SetTimeout(t int)
	GetTimeout() int
	Reconnect(t int) //重连间隔  最小1s，此函数仅能连接关闭是调用

	GetHandler() IMsgHandler

	SetUser(user interface{})
	User() interface{}

	GetSendCallback(tag int) chan interface{}
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

	init     bool
	callback map[int]chan interface{}
	user     interface{}
}

func (r *msgQue) SetUser(user interface{}) {
	r.user = user
}

func (r *msgQue) GetSendCallback(tag int) chan interface{} {
	if r.callback != nil {
		if c, ok := r.callback[tag]; ok {
			return c
		}
	}
	return nil
}

func (r *msgQue) SetCallback(tag int, c chan interface{}) {
	if r.callback == nil {
		r.callback = make(map[int]chan interface{})
	}
	oc, ok := r.callback[tag]
	if ok {
		oc <- nil
	}

	r.callback[tag] = c
}

func (r *msgQue) User() interface{} {
	return r.user
}

func (r *msgQue) GetHandler() IMsgHandler {
	return r.handler
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
	if msgque.GetConnType() == ConnTypeConn {
		if c := msgque.GetSendCallback(msg.Tag()); c != nil {
			c <- msg
		}
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
	if addrs[0] == "udp" {
		naddr, err := net.ResolveUDPAddr(addrs[0], addrs[1])
		if err != nil {
			return err
		}
		conn, err := net.ListenUDP(addrs[0], naddr)
		if err != nil {
			return err
		}
		msgque := newUdpListen(conn, typ, handler, parser, addr)
		Go(func() {
			LogDebug("process listen for msgque:%d", msgque.id)
			msgque.listen()
			LogDebug("process listen end for msgque:%d", msgque.id)
		})

	} else {
		listen, err := net.Listen(addrs[0], addrs[1])
		if err == nil {
			msgque := newTcpListen(listen, typ, handler, parser, addr)
			Go(func() {
				LogDebug("process listen for msgque:%d", msgque.id)
				msgque.listen()
				LogDebug("process listen end for msgque:%d", msgque.id)
			})
		} else {
			LogError(err)
		}
		return err
	}
	return nil
}

func StartConnect(netype string, addr string, typ MsgType, handler IMsgHandler, parser *Parser, user interface{}) {
	msgque := newTcpConn(netype, addr, nil, typ, handler, parser, user)
	if handler.OnNewMsgQue(msgque) {
		msgque.Reconnect(0)
	} else {
		msgque.Stop()
	}
}
