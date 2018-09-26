package antnet

import "encoding/json"

type JsonParser struct {
	*Parser
}

func (r *JsonParser) ParseC2S(msg *Message) (IMsgParser, error) {
	if msg == nil {
		return nil, ErrJsonUnPack
	}

	if msg.Head == nil {
		if len(msg.Data) == 0 {
			return nil, ErrJsonUnPack
		}
		for _, p := range r.typMap {
			if p.C2S() != nil {
				err := JsonUnPack(msg.Data, p.C2S())
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
				err := JsonUnPack(msg.Data, p.C2S())
				if err != nil {
					return nil, err
				}
			}
			p.parser = r
			return &p, nil
		}
	}

	return nil, ErrJsonUnPack
}

func (r *JsonParser) PackMsg(v interface{}) []byte {
	data, _ := JsonPack(v)
	return data
}

func (r *JsonParser) GetRemindMsg(err error, t MsgType) *Message {
	if t == MsgTypeMsg {
		return NewErrMsg(err)
	} else {
		return NewStrMsg(err.Error() + "\n")
	}
}

func JsonUnPack(data []byte, msg interface{}) error {
	if data == nil || msg == nil {
		return ErrJsonUnPack
	}

	err := json.Unmarshal(data, msg)
	if err != nil {
		return ErrJsonUnPack
	}
	return nil
}

func JsonPack(msg interface{}) ([]byte, error) {
	if msg == nil {
		return nil, ErrJsonPack
	}

	data, err := json.Marshal(msg)
	if err != nil {
		LogInfo("")
		return nil, ErrJsonPack
	}

	return data, nil
}
