package antnet

import (
	"net"
	"reflect"
	"strings"
	"sync"
	"time"
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
	NetTypeWs                 //websocket
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
	SetRealRemoteAddr(addr string)

	Stop()
	IsStop() bool
	Available() bool
	IsProxy() bool

	Send(m *Message) (re bool)
	SendString(str string) (re bool)
	SendStringLn(str string) (re bool)
	SendByteStr(str []byte) (re bool)
	SendByteStrLn(str []byte) (re bool)
	SendCallback(m *Message, c chan *Message) (re bool)
	DelCallback(m *Message)
	SetSendFast()
	SetTimeout(t int)
	SetCmdReadRaw()
	GetTimeout() int
	Reconnect(t int) //重连间隔  最小1s，此函数仅能连接关闭是调用

	GetHandler() IMsgHandler

	SetUser(user interface{})
	GetUser() interface{}

	SetGroupId(group string)
	DelGroupId(group string)
	ClearGroupId(group string)
	IsInGroup(group string) bool

	SetMultiplex(multiplex bool) bool

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
	parserFactory IParserFactory
	timeout       int //传输超时
	lastTick      int64

	init           bool
	available      bool
	sendFast       bool
	multiplex      bool
	callback       map[int]chan *Message
	group          map[string]int
	user           interface{}
	callbackLock   sync.Mutex
	gmsgId         uint16
	realRemoteAddr string //当使用代理是，需要特殊设置客户端真实IP
}

func (r *msgQue) SetSendFast() {
	r.sendFast = true
}

func (r *msgQue) SetUser(user interface{}) {
	r.user = user
}

func (r *msgQue) getGMsg(add bool) *gMsg {
	if add {
		r.gmsgId++
	}
	gm := gmsgArray[r.gmsgId]
	return gm
}
func (r *msgQue) SetCmdReadRaw() {

}
func (r *msgQue) Available() bool {
	return r.available
}

func (r *msgQue) GetUser() interface{} {
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
	if t >= 0 {
		r.timeout = t
	}
}

func (r *msgQue) isTimeout(tick *time.Timer) bool {
	left := int(Timestamp - r.lastTick)
	if left < r.timeout || r.timeout == 0 {
		if r.timeout == 0 {
			tick.Reset(time.Second * time.Duration(DefMsgQueTimeout))
		} else {
			tick.Reset(time.Second * time.Duration(r.timeout-left))
		}
		return false
	}
	LogInfo("msgque close because timeout id:%v wait:%v timeout:%v", r.id, left, r.timeout)
	return true
}

func (r *msgQue) GetTimeout() int {
	return r.timeout
}

func (r *msgQue) Reconnect(t int) {

}

func (r *msgQue) IsProxy() bool{
	return r.realRemoteAddr != ""
}

func (r *msgQue) SetRealRemoteAddr(addr string) {
	r.realRemoteAddr = addr
}

func (r *msgQue) SetGroupId(group string) {
	r.callbackLock.Lock()
	if r.group == nil {
		r.group = make(map[string]int)
	}
	r.group[group] = 0
	r.callbackLock.Unlock()
}

func (r *msgQue) DelGroupId(group string) {
	r.callbackLock.Lock()
	if r.group != nil {
		delete(r.group, group)
	}
	r.callbackLock.Unlock()
}

func (r *msgQue) ClearGroupId(group string) {
	r.callbackLock.Lock()
	r.group = nil
	r.callbackLock.Unlock()
}

func (r *msgQue) IsInGroup(group string) bool {
	re := false
	r.callbackLock.Lock()
	if r.group != nil {
		_, re = r.group[group]
	}
	r.callbackLock.Unlock()
	return re
}

func (r *msgQue) SetMultiplex(multiplex bool) bool {
	t := r.multiplex
	r.multiplex = multiplex
	return t
}

func (r *msgQue) Send(m *Message) (re bool) {
	if m == nil || !r.available {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			re = false
		}
	}()
	if Config.AutoCompressLen > 0 && m.Head != nil && m.Head.Len >= Config.AutoCompressLen && (m.Head.Flags&FlagCompress) == 0 {
		m.Head.Flags |= FlagCompress
		m.Data = GZipCompress(m.Data)
		m.Head.Len = uint32(len(m.Data))
	}
	select {
	case r.cwrite <- m:
	default:
		LogWarn("msgque write channel full msgque:%v", r.id)
		r.cwrite <- m
	}

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

func (r *msgQue) DelCallback(m *Message) {
	if r.callback == nil {
		return
	}
	r.callbackLock.Lock()
	delete(r.callback, m.Tag())
	r.callbackLock.Unlock()
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

func (r *msgQue) baseStop() {
	if r.cwrite != nil {
		close(r.cwrite)
	}

	for k, v := range r.callback {
		Try(func() {
			v <- nil
		}, func(i interface{}) {

		})
		delete(r.callback, k)
	}
	msgqueMapSync.Lock()
	delete(msgqueMap, r.id)
	msgqueMapSync.Unlock()
	LogInfo("msgque close id:%d", r.id)
}
func (r *msgQue) processMsg(msgque IMsgQue, msg *Message) bool {
	if r.multiplex {
		Go(func() {
			r.processMsgTrue(msgque, msg)
		})
	} else {
		return r.processMsgTrue(msgque, msg)
	}
	return true
}
func (r *msgQue) processMsgTrue(msgque IMsgQue, msg *Message) bool {
	if msg.Head != nil && msg.Head.Flags&FlagCompress > 0 && msg.Data != nil {
		data, err := GZipUnCompress(msg.Data)
		if err != nil {
			LogError("msgque uncompress failed msgque:%v cmd:%v act:%v len:%v err:%v", msgque.Id(), msg.Head.Cmd, msg.Head.Act, msg.Head.Len, err)
			return false
		}
		msg.Data = data
		msg.Head.Flags -= FlagCompress
		msg.Head.Len = uint32(len(msg.Data))
	}
	if r.parser != nil {
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
	if msgque.tryCallback(msg) {
		return true
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

type DefMsgHandler struct {
	msgMap  map[int]HandlerFunc
	typeMap map[reflect.Type]HandlerFunc
}

func (r *DefMsgHandler) OnNewMsgQue(msgque IMsgQue) bool                { return true }
func (r *DefMsgHandler) OnDelMsgQue(msgque IMsgQue)                     {}
func (r *DefMsgHandler) OnProcessMsg(msgque IMsgQue, msg *Message) bool { return true }
func (r *DefMsgHandler) OnConnectComplete(msgque IMsgQue, ok bool) bool { return true }
func (r *DefMsgHandler) GetHandlerFunc(msgque IMsgQue, msg *Message) HandlerFunc {
	if msg.Head == nil {
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

func StartServer(addr string, typ MsgType, handler IMsgHandler, parser IParserFactory) error {
	addrs := strings.Split(addr, "://")
	if addrs[0] == "tcp" || addrs[0] == "all" {
		listen, err := net.Listen("tcp", addrs[1])
		if err == nil {
			msgque := newTcpListen(listen, typ, handler, parser, addr)
			Go(func() {
				LogDebug("process listen for tcp msgque:%d", msgque.id)
				msgque.listen()
				LogDebug("process listen end for tcp msgque:%d", msgque.id)
			})
		} else {
			LogError("listen on %s failed, errstr:%s", addr, err)
			return err
		}
	}
	if addrs[0] == "udp" || addrs[0] == "all" {
		naddr, err := net.ResolveUDPAddr("udp", addrs[1])
		if err != nil {
			LogError("listen on %s failed, errstr:%s", addr, err)
			return err
		}
		conn, err := net.ListenUDP("udp", naddr)
		if err == nil {
			msgque := newUdpListen(conn, typ, handler, parser, addr)
			Go(func() {
				LogDebug("process listen for udp msgque:%d", msgque.id)
				msgque.listen()
				LogDebug("process listen end for udp msgque:%d", msgque.id)
			})
		} else {
			LogError("listen on %s failed, errstr:%s", addr, err)
			return err
		}
	}
	if addrs[0] == "ws" || addrs[0] == "wss" {
		naddr := strings.SplitN(addrs[1], "/", 2)
		url := "/"
		if len(naddr) > 1 {
			url = "/" + naddr[1]
		}
		if addrs[0] == "wss" {
			Config.EnableWss = true
		}
		if typ != MsgTypeCmd {
			LogInfo("ws type msgque noly support MsgTypeCmd now auto set to MsgTypeCmd")
		}
		msgque := newWsListen(naddr[0], url, MsgTypeCmd, handler, parser)
		Go(func() {
			LogDebug("process listen for ws msgque:%d", msgque.id)
			msgque.listen()
			LogDebug("process listen end for ws msgque:%d", msgque.id)
		})
	}
	return nil
}

func StartConnect(netype string, addr string, typ MsgType, handler IMsgHandler, parser IParserFactory, user interface{}) IMsgQue {
	msgque := newTcpConn(netype, addr, nil, typ, handler, parser, user)
	if handler.OnNewMsgQue(msgque) {
		msgque.Reconnect(0)
		return msgque
	} else {
		msgque.Stop()
	}
	return nil
}
