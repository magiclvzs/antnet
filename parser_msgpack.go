package antnet

import (
	"github.com/vmihailenco/msgpack"
)

type MsgpackParser struct {
	*Parser
}

func (r *MsgpackParser) ParseC2S(msg *Message) (IMsgParser, error) {
	if msg == nil {
		return nil, ErrMsgPackUnPack
	}

	if msg.Head == nil {
		if len(msg.Data) == 0 {
			return nil, ErrMsgPackUnPack
		}
		for _, p := range r.typMap {
			if p.C2S() != nil {
				err := MsgPackUnPack(msg.Data, p.C2S())
				if err != nil {
					continue
				}
				p.parser = r
				return &p, nil
			}
		}
	} else if p, ok := r.msgMap[msg.Head.CmdAct()]; ok {
		if p.C2S() != nil {
			if len(msg.Data) > 0 {
				err := MsgPackUnPack(msg.Data, p.C2S())
				if err != nil {
					return nil, err
				}
			}
			p.parser = r
			return &p, nil
		}
	}

	return nil, ErrMsgPackUnPack
}

func (r *MsgpackParser) PackMsg(v interface{}) []byte {
	data, _ := MsgPackPack(v)
	return data
}

func (r *MsgpackParser) GetRemindMsg(err error, t MsgType) *Message {
	if t == MsgTypeMsg {
		return NewErrMsg(err)
	} else {
		return NewStrMsg(err.Error() + "\n")
	}
}

func MsgPackUnPack(data []byte, msg interface{}) error {
	err := msgpack.Unmarshal(data, msg)
	return err
}

func MsgPackPack(msg interface{}) ([]byte, error) {
	data, err := msgpack.Marshal(msg)
	return data, err
}
