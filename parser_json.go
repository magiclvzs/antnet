package antnet

import (
	"encoding/json"
	"reflect"
)

type jsonParser struct {
	factory *Parser
}

type jsonParseNode struct {
	c2sFunc ParseFunc
	s2cFunc ParseFunc
}

func (r *jsonParser) ParseC2S(msg *Message) (IMsgParser, error) {
	if msg == nil {
		return nil, ErrJsonUnPack
	}

	if msg.Head == nil {
		m := map[string]json.RawMessage{}
		if json.Unmarshal(msg.Data, &m) == nil {
			for k, v := range m {
				node, ok := r.factory.jsonMap[k]
				if ok {
					c2s := node.c2sFunc()
					if json.Unmarshal(v, c2s) == nil {
						return &MsgParser{c2s: c2s, parser: r, s2cFunc: node.s2cFunc}, nil
					}
				}
			}
		}
	} else {
		p, ok := r.factory.msgMap[msg.Head.CmdAct()]
		if ok {
			if p.C2S() != nil {
				err := JsonUnPack(msg.Data, p.C2S())
				if err != nil {
					return nil, err
				}
				p.parser = r
				return &p, nil
			}
		}
	}
	return nil, ErrJsonUnPack
}

func (r *jsonParser) PackMsg(v interface{}) []byte {
	data, _ := JsonPack(v)
	return data
}

func (r *jsonParser) GetRemindMsg(err error, t MsgType) *Message {
	if t == MsgTypeMsg {
		return NewErrMsg(err)
	} else {
		return NewStrMsg(err.Error() + "\n")
	}
}

func (r *jsonParser) GetType() ParserType {
	return r.factory.Type
}

func (r *jsonParser) GetErrType() ParseErrType {
	return r.factory.ErrType
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

func registerJsonParser(jm map[string]*jsonParseNode, c2sFunc ParseFunc, s2cFunc ParseFunc) {
	msgType := reflect.TypeOf(c2sFunc())
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		LogFatal("message pointer required")
		return
	}
	typ := msgType.Elem()
	jm[typ.Name()] = &jsonParseNode{c2sFunc, s2cFunc}
}
