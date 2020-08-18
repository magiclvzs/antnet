package antnet

import (
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/go-redis/redis"
)

type RedisConfig struct {
	Addr     string
	Passwd   string
	PoolSize int
}

type Redis struct {
	*redis.Client
	pubsub  *redis.PubSub
	conf    *RedisConfig
	manager *RedisManager
}

func (r *Redis) ScriptStr(cmd int, keys []string, args ...interface{}) (string, error) {
	data, err := r.Script(cmd, keys, args...)
	if err != nil {
		LogError("redis script failed err:%v", err)
		return "", ErrDBErr
	}
	errcode, ok := data.(int64)
	if ok {
		return "", GetError(uint16(errcode))
	}
	if data == nil {
		return "", nil
	}
	str, ok := data.(string)
	if !ok {
		return "", ErrDBDataType
	}

	return str, nil
}

func (r *Redis) ScriptStrArray(cmd int, keys []string, args ...interface{}) ([]string, error) {
	data, err := r.Script(cmd, keys, args...)
	if err != nil {
		LogError("redis script failed err:%v", err)
		return nil, ErrDBErr
	}
	errcode, ok := data.(int64)
	if ok {
		return nil, GetError(uint16(errcode))
	}

	iArray, ok := data.([]interface{})
	if !ok {
		return nil, ErrDBDataType
	}

	strArray := []string{}
	for _, v := range iArray {
		if v == nil {
			strArray = append(strArray, "")
		} else if str, ok := v.(string); ok {
			strArray = append(strArray, str)
		} else {
			return nil, ErrDBDataType
		}
	}

	return strArray, nil
}

func (r *Redis) ScriptInt64(cmd int, keys []string, args ...interface{}) (int64, error) {
	data, err := r.Script(cmd, keys, args...)
	if err != nil {
		LogError("redis script failed err:%v", err)
		return 0, ErrDBErr
	}
	if data == nil {
		return 0, nil
	}
	code, ok := data.(int64)
	if ok {
		return code, nil
	}
	return 0, ErrDBDataType
}

func (r *Redis) Script(cmd int, keys []string, args ...interface{}) (interface{}, error) {
	var err error = ErrDBErr
	var re interface{}
	hashStr, ok := scriptHashMap.Load(cmd)
	if ok {
		re, err = r.EvalSha(hashStr.(string), keys, args...).Result()
	}
	if RedisError(err) {
		scriptStr, ok := scriptMap.Load(cmd)
		if !ok {
			LogError("redis script error cmd not found cmd:%v", cmd)
			return nil, ErrDBErr
		}
		cmdStr, _ := scriptCommitMap.Load(cmd)
		if strings.HasPrefix(err.Error(), "NOSCRIPT ") || err == ErrDBErr {
			LogInfo("try reload redis script %v", cmdStr.(string))
			hashStr, err = r.ScriptLoad(scriptStr.(string)).Result()
			if err != nil {
				LogError("redis script load cmd:%v errstr:%s", cmdStr.(string), err)
				return nil, ErrDBErr
			}
			scriptHashMap.Store(cmd, hashStr.(string))
			re, err = r.EvalSha(hashStr.(string), keys, args...).Result()
			if !RedisError(err) {
				return re, nil
			}
		}
		LogError("redis script error cmd:%v errstr:%s", cmdStr.(string), err)
		return nil, ErrDBErr
	}

	return re, nil
}

type RedisManager struct {
	dbs      map[int]*Redis
	subMap   map[string]*Redis
	channels []string
	fun      func(channel, data string)
	lock     sync.RWMutex
}

func (r *RedisManager) GetByRid(rid int) *Redis {
	r.lock.RLock()
	db := r.dbs[rid]
	r.lock.RUnlock()
	return db
}

func (r *RedisManager) GetGlobal() *Redis {
	return r.GetByRid(0)
}

func (r *RedisManager) Sub(fun func(channel, data string), channels ...string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.channels = channels
	r.fun = fun
	for _, v := range r.subMap {
		if v.pubsub != nil {
			v.pubsub.Close()
		}
		pubsub := v.Subscribe(channels...)
		v.pubsub = pubsub
		LogInfo("[redis]config:%v, subscribe channel:%v", v.conf, channels)
		goForRedis(func() {
			for IsRuning() {
				msg, err := pubsub.ReceiveMessage()
				if err == nil {
					Go(func() { fun(msg.Channel, msg.Payload) })
				} else if _, ok := err.(net.Error); !ok {
					if err.Error() != "redis: reply is empty" {
						LogFatal("[redis]pubsub broken err:%v", err)
						break
					}
				}
			}
		})
	}
}

func (r *RedisManager) Exist(id int) bool {
	r.lock.Lock()
	_, ok := r.dbs[id]
	r.lock.Unlock()
	return ok
}

func (r *RedisManager) Add(id int, conf *RedisConfig) {
	r.lock.Lock()
	defer r.lock.Unlock()
	LogInfo("new redis id:%v conf:%#v", id, conf)
	if _, ok := r.dbs[id]; ok {
		LogError("redis already have id:%v", id)
		return
	}

	re := &Redis{
		Client: redis.NewClient(&redis.Options{
			Addr:     conf.Addr,
			Password: conf.Passwd,
			PoolSize: conf.PoolSize,
		}),
		conf:    conf,
		manager: r,
	}

	re.WrapProcess(func(oldProcess func(cmd redis.Cmder) error) func(cmd redis.Cmder) error {
		return func(cmd redis.Cmder) error {
			err := oldProcess(cmd)
			if err != nil {
				_, retry := err.(net.Error)
				if !retry {
					retry = err == io.EOF
				}
				if retry {
					err = oldProcess(cmd)
				}
			}
			return err
		}
	})

	if v, ok := r.subMap[conf.Addr]; !ok {
		r.subMap[conf.Addr] = re
		if len(r.channels) > 0 {
			pubsub := re.Subscribe(r.channels...)
			re.pubsub = pubsub
			LogInfo("[redis]config:%v, subscribe channel:%v", v.conf, r.channels)
			goForRedis(func() {
				for IsRuning() {
					msg, err := pubsub.ReceiveMessage()
					if err == nil {
						Go(func() { r.fun(msg.Channel, msg.Payload) })
					} else if _, ok := err.(net.Error); !ok && IsRuning() {
						if err.Error() != "redis: reply is empty" {
							LogFatal("[redis]pubsub broken err:%v", err)
							break
						}
					}
				}
			})
		}
	}
	r.dbs[id] = re
}

func (r *RedisManager) close() {
	for _, v := range r.dbs {
		if v.pubsub != nil {
			v.pubsub.Close()
		}
		v.Close()
	}
}

var (
	scriptMap             = sync.Map{} //map[int]string{}
	scriptCommitMap       = sync.Map{} //map[int]string{}
	scriptHashMap         = sync.Map{} //map[int]string{}
	scriptIndex     int32 = 0
)

func NewRedisScript(commit, str string) int {
	cmd := int(atomic.AddInt32(&scriptIndex, 1))
	scriptMap.Store(cmd, str)
	scriptCommitMap.Store(cmd, commit)
	return cmd
}

func GetRedisScript(cmd int) string {
	if s, ok := scriptMap.Load(cmd); ok {
		return s.(string)
	}
	return ""
}

var redisManagers []*RedisManager

func NewRedisManager(conf *RedisConfig) *RedisManager {
	redisManager := &RedisManager{
		subMap: map[string]*Redis{},
		dbs:    map[int]*Redis{},
	}

	redisManager.Add(0, conf)
	redisManagers = append(redisManagers, redisManager)
	return redisManager
}

func RedisError(err error) bool {
	if err == redis.Nil {
		return false
	}
	return err != nil
}
