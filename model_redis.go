// 模型来自pb
// 特别注意，lua只至此double，int64的数据如果进行cmsgpack打包解包可能出现精度问题导致bug
package antnet

import (
	"github.com/golang/protobuf/proto"
	"github.com/vmihailenco/msgpack"
)

type RedisModel struct{}

func (r *RedisModel) DBData(v proto.Message) []byte {
	return DBData(v)
}

func (r *RedisModel) DBStr(v proto.Message) string {
	return DBStr(v)
}

func (r *RedisModel) PbData(v proto.Message) []byte {
	return PbData(v)
}

func (r *RedisModel) PbStr(v proto.Message) string {
	return PbStr(v)
}

func (r *RedisModel) ParseDBData(data []byte, v proto.Message) bool {
	return ParseDBData(data, v)
}

func (r *RedisModel) ParseDBStr(str string, v proto.Message) bool {
	return ParseDBStr(str, v)
}

func (r *RedisModel) ParsePbData(data []byte, v proto.Message) bool {
	return ParsePbData(data, v)
}

func (r *RedisModel) ParsePbStr(str string, v proto.Message) bool {
	return ParsePbStr(str, v)
}

func DBData(v proto.Message) []byte {
	data, err := msgpack.Marshal(v)
	if err != nil {
		LogError(err)
	}
	return data
}

func DBStr(v proto.Message) string {
	data, err := msgpack.Marshal(v)
	if err != nil {
		LogError(err)
	}
	return string(data)
}

func PbData(v proto.Message) []byte {
	data, err := proto.Marshal(v)
	if err != nil {
		LogError(err)
	}
	return data
}

func PbStr(v proto.Message) string {
	data, err := proto.Marshal(v)
	if err != nil {
		LogError(err)
	}
	return string(data)
}

func ParseDBData(data []byte, v proto.Message) bool {
	if err := msgpack.Unmarshal(data, v); err != nil {
		return ParsePbData(data, v)
	}
	return true
}

func ParseDBStr(str string, v proto.Message) bool {
	if err := msgpack.Unmarshal([]byte(str), v); err != nil {
		return ParsePbStr(str, v)
	}
	return true
}

func ParsePbData(data []byte, v proto.Message) bool {
	if err := proto.Unmarshal(data, v); err != nil {
		LogError(err)
		return false
	}
	return true
}

func ParsePbStr(str string, v proto.Message) bool {
	if err := proto.Unmarshal([]byte(str), v); err != nil {
		LogError(err)
		return false
	}
	return true
}
