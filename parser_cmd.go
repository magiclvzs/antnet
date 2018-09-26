package antnet

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
)

type CmdMatchType int

const (
	CmdMatchTypeK CmdMatchType = iota
	CmdMatchTypeKV
)

type cmdParseNode struct {
	match   CmdMatchType
	kind    reflect.Kind
	index   int
	name    string
	c2sFunc ParseFunc
	s2cFunc ParseFunc
	next    map[string]*cmdParseNode
	prev    *cmdParseNode
}
type CmdParser struct {
	*Parser
	node   *cmdParseNode
	values []interface{}
	match  CmdMatchType
}

func (r *CmdParser) ParseC2S(msg *Message) (IMsgParser, error) {
	p, ok := r.parserString(string(msg.Data))
	if ok {
		return p, nil
	}

	return nil, ErrCmdUnPack
}

func (r *CmdParser) PackMsg(v interface{}) []byte {
	data, _ := JsonPack(v)
	if data != nil {
		var out bytes.Buffer
		err := json.Indent(&out, data, "", "\t")
		if err == nil {
			return out.Bytes()
		}
	}
	return data
}

func (r *CmdParser) GetRemindMsg(err error, t MsgType) *Message {
	if t == MsgTypeMsg {
		return nil
	} else {
		return nil
	}
}

func (r *CmdParser) parserString(s string) (IMsgParser, bool) {
	if r.node == nil {
		r.node = r.cmdRoot
	}
	s = strings.TrimSpace(s)
	cmds := strings.Split(s, " ")

	for _, v := range cmds {
		if r.match == CmdMatchTypeK {
			node, ok := r.node.next[v]
			if !ok {
				return nil, false
			}
			r.node = node
			if node.match == CmdMatchTypeK {
				continue
			} else {
				r.match = CmdMatchTypeKV
				continue
			}
		}
		i, err := ParseBaseKind(r.node.kind, v)
		if err != nil {
			return nil, false
		}

		r.match = CmdMatchTypeK
		r.values = append(r.values, i)
	}

	if r.node.next == nil {
		typ := reflect.ValueOf(r.node.c2sFunc())
		ins := typ.Elem()
		i := len(r.values)
		s2cFunc := r.node.s2cFunc
		for r.node != nil {
			if r.node.match == CmdMatchTypeKV {
				ins.Field(r.node.index).Set(reflect.ValueOf(r.values[i-1]))
				i--
			} else if r.node.kind == reflect.String && r.node != r.cmdRoot {
				ins.Field(r.node.index).SetString(r.node.name)
			}
			r.node = r.node.prev
		}
		return &MsgParser{c2s: typ.Interface(), parser: r, s2cFunc: s2cFunc}, true
	}
	return nil, false
}

func registerCmdParser(root *cmdParseNode, c2sFunc ParseFunc, s2cFunc ParseFunc) {
	msgType := reflect.TypeOf(c2sFunc())
	if msgType == nil || msgType.Kind() != reflect.Ptr {
		LogFatal("message pointer required")
		return
	}
	typ := msgType.Elem()
	prevRoute := root

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := strings.ToLower(field.Name)
		tag := field.Tag
		if prevRoute.next == nil {
			prevRoute.next = map[string]*cmdParseNode{}
		}
		c, ok := prevRoute.next[name]
		if !ok {
			c = &cmdParseNode{}
			c.prev = prevRoute

			if tag.Get("match") == "k" {
				c.match = CmdMatchTypeK
			} else {
				c.match = CmdMatchTypeKV
			}
			c.name = name
			c.kind = field.Type.Kind()
			c.index = i
			prevRoute.next[name] = c
		}
		prevRoute = c
	}

	prevRoute.s2cFunc = s2cFunc
	prevRoute.c2sFunc = c2sFunc
}
