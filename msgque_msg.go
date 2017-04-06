package antnet

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	MsgHeadSize = 12
)

const (
	FlagEncrypt  = 1 << 0 //数据是经过加密的
	FlagCompress = 1 << 1 //数据是经过压缩的
	FlagContinue = 1 << 2 //消息还有后续
	FlagNeedAck  = 1 << 3 //消息需要确认
	FlagAck      = 1 << 4 //确认消息
	FlagReSend   = 1 << 5 //重发消息
	FlagClient   = 1 << 6 //消息来自客服端，用于判断index来之服务器还是其他玩家
)

var MaxMsgDataSize uint32 = 1024 * 1024

type MessageHead struct {
	Len   uint32 //数据长度
	Error uint16 //错误码
	Cmd   uint8  //命令
	Act   uint8  //动作
	Index uint16 //序号
	Flags uint16 //标记
}

func (r *MessageHead) Bytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.Grow(MsgHeadSize)
	typ := binary.LittleEndian
	binary.Write(buf, typ, r.Len)
	binary.Write(buf, typ, r.Error)
	binary.Write(buf, typ, r.Cmd)
	binary.Write(buf, typ, r.Act)
	binary.Write(buf, typ, r.Index)
	binary.Write(buf, typ, r.Flags)
	return buf.Bytes()
}

func (r *MessageHead) FromBytes(data []byte) error {
	buf := bytes.NewBuffer(data)
	typ := binary.LittleEndian
	binary.Read(buf, typ, &r.Len)
	binary.Read(buf, typ, &r.Error)
	binary.Read(buf, typ, &r.Cmd)
	binary.Read(buf, typ, &r.Act)
	binary.Read(buf, typ, &r.Index)
	binary.Read(buf, typ, &r.Flags)
	if r.Len > MaxMsgDataSize {
		return errors.New("head len too big")
	}
	return nil
}

func (r *MessageHead) CmdAct() int {
	return CmdAct(r.Cmd, r.Act)
}

func (r *MessageHead) Tag() int {
	return Tag(r.Cmd, r.Act, r.Index)
}

func (r *MessageHead) String() string {
	return fmt.Sprintf("Len:%v Error:%v Cmd:%v Act:%v Index:%v Flags:%v", r.Len, r.Error, r.Cmd, r.Act, r.Index, r.Flags)
}

func NewMessageHead(data []byte) *MessageHead {
	head := MessageHead{}
	if err := head.FromBytes(data); err != nil {
		return nil
	}
	return &head
}

type Message struct {
	Head       *MessageHead //消息头，可能为nil
	Data       []byte       //消息数据
	IMsgParser              //解析器
	User       interface{}  //用户自定义数据
}

func (r *Message) CmdAct() int {
	if r.Head != nil {
		return CmdAct(r.Head.Cmd, r.Head.Act)
	}
	return 0
}

func (r *Message) Tag() int {
	if r.Head != nil {
		return Tag(r.Head.Cmd, r.Head.Act, r.Head.Index)
	}
	return 0
}

func (r *Message) CopyTag(old *Message) *Message {
	r.Head.Cmd = old.Head.Cmd
	r.Head.Act = old.Head.Act
	r.Head.Index = old.Head.Index
	return r
}

func NewErrMsg(err error) *Message {
	errcode, ok := errIdMap[err]
	if !ok {
		errcode = errIdMap[ErrErrIdNotFound]
	}
	return &Message{
		Head: &MessageHead{
			Error: errcode,
		},
	}
}

func NewStrMsg(str string) *Message {
	return &Message{
		Data: []byte(str),
	}
}

func NewDataMsg(data []byte) *Message {
	return &Message{
		Head: &MessageHead{
			Len: uint32(len(data)),
		},
		Data: data,
	}
}

func NewMsg(cmd, act uint8, index, err uint16, data []byte) *Message {
	return &Message{
		Head: &MessageHead{
			Len:   uint32(len(data)),
			Error: err,
			Cmd:   cmd,
			Act:   act,
			Index: index,
		},
		Data: data,
	}
}

func NewTagMsg(cmd, act uint8, index uint16) *Message {
	return &Message{
		Head: &MessageHead{
			Cmd:   cmd,
			Act:   act,
			Index: index,
		},
	}
}
