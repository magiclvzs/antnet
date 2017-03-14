//模型来自pb
package antnet

import (
	"github.com/golang/protobuf/proto"
	"gopkg.in/vmihailenco/msgpack.v2"
)

type RedisModel struct{}

func (r *RedisModel) DBData(v proto.Message) []byte {
	data, _ := msgpack.Marshal(v)
	return data
}

func (r *RedisModel) DBDataStr(v proto.Message) string {
	data, _ := msgpack.Marshal(v)
	return string(data)
}

func (r *RedisModel) PbData(v proto.Message) []byte {
	data, _ := proto.Marshal(v)
	return data
}

func (r *RedisModel) ParseDB(data []byte, v proto.Message) bool {
	err := msgpack.Unmarshal(data, v)
	return err == nil
}

func (r *RedisModel) ParseDBStr(str string, v proto.Message) bool {
	err := msgpack.Unmarshal([]byte(str), v)
	return err == nil
}

func (r *RedisModel) ParseDBInf(inf interface{}, v proto.Message) bool {
	err := msgpack.Unmarshal([]byte(inf.(string)), v)
	return err == nil
}

func (r *RedisModel) ParsePb(data []byte, v proto.Message) bool {
	err := proto.Unmarshal(data, v)
	return err == nil
}
