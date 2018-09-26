package antnet

import (
	"github.com/golang/protobuf/proto"
)

type PBParser struct {
	*Parser
}

func (r *PBParser) ParseC2S(msg *Message) (IMsgParser, error) {
	if msg == nil {
		return nil, ErrPBUnPack
	}

	if msg.Head == nil {
		if len(msg.Data) == 0 {
			return nil, ErrPBUnPack
		}
		for _, p := range r.typMap {
			if p.C2S() != nil {
				err := PBUnPack(msg.Data, p.C2S())
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
				err := PBUnPack(msg.Data, p.C2S())
				if err != nil {
					return nil, err
				}
			}
			p.parser = r
			return &p, nil
		}
	}

	return nil, ErrPBUnPack
}

func (r *PBParser) PackMsg(v interface{}) []byte {
	data, _ := PBPack(v)
	return data
}

func (r *PBParser) GetRemindMsg(err error, t MsgType) *Message {
	if t == MsgTypeMsg {
		return NewErrMsg(err)
	} else {
		return NewStrMsg(err.Error() + "\n")
	}
}

func PBUnPack(data []byte, msg interface{}) error {
	if data == nil || msg == nil {
		return ErrPBUnPack
	}

	err := proto.Unmarshal(data, msg.(proto.Message))
	if err != nil {
		return ErrPBUnPack
	}
	return nil
}

func PBPack(msg interface{}) ([]byte, error) {
	if msg == nil {
		return nil, ErrPBPack
	}

	data, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		LogInfo("")
	}

	return data, nil
}
