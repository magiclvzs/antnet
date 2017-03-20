//模型来自pb
package antnet

import (
	"github.com/golang/protobuf/proto"
	"gopkg.in/vmihailenco/msgpack.v2"
)

type RedisModel struct{}

func (r *RedisModel) DBData(v interface{}) []byte {
	return DBData(v)
}

func (r *RedisModel) DBStr(v interface{}) string {
	return DBStr(v)
}

func (r *RedisModel) PbData(v proto.Message) []byte {
	return PbData(v)
}

func (r *RedisModel) ParseDBData(data []byte, v interface{}) bool {
	return ParseDBData(data, v)
}

func (r *RedisModel) ParseDBStr(str string, v interface{}) bool {
	return ParseDBStr(str, v)
}

func (r *RedisModel) ParsePb(data []byte, v proto.Message) bool {
	return ParsePb(data, v)
}

func DBData(v interface{}) []byte {
	data, _ := msgpack.Marshal(v)
	return data
}

func DBStr(v interface{}) string {
	data, _ := msgpack.Marshal(v)
	return string(data)
}

func PbData(v proto.Message) []byte {
	data, _ := proto.Marshal(v)
	return data
}

func ParseDBData(data []byte, v interface{}) bool {
	err := msgpack.Unmarshal(data, v)
	return err == nil
}

func ParseDBStr(str string, v interface{}) bool {
	err := msgpack.Unmarshal([]byte(str), v)
	return err == nil
}

func ParsePb(data []byte, v proto.Message) bool {
	err := proto.Unmarshal(data, v)
	return err == nil
}
