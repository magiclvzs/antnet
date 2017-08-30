package antnet

import (
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
		return "", err
	}
	errcode, ok := data.(int64)
	if ok {
		return "", GetError(uint16(errcode))
	}

	str, ok := data.(string)
	if !ok {
		return "", ErrDBDataType
	}

	return str, nil
}

func (r *Redis) ScriptInt64(cmd int, keys []string, args ...interface{}) (int64, error) {
	data, err := r.Script(cmd, keys, args...)
	if err != nil {
		return 0, err
	}
	code, ok := data.(int64)
	if ok {
		return code, nil
	}
	return 0, ErrDBDataType
}

func (r *Redis) Script(cmd int, keys []string, args ...interface{}) (interface{}, error) {
	hash, _ := scriptHashMap[cmd]
	re, err := r.EvalSha(hash, keys, args...).Result()
	if err != nil {
		script, ok := scriptMap[cmd]
		if !ok {
			LogError("redis script error cmd not found cmd:%v", cmd)
			return nil, ErrDBErr
		}

		if strings.HasPrefix(err.Error(), "NOSCRIPT ") {
			LogInfo("try reload redis script %v", scriptCommitMap[cmd])
			hash, err = r.ScriptLoad(script).Result()
			if err != nil {
				LogError("redis script load cmd:%v errstr:%s", scriptCommitMap[cmd], err)
				return nil, ErrDBErr
			}
			scriptHashMap[cmd] = hash
			re, err = r.EvalSha(hash, keys, args...).Result()
			if err == nil {
				return re, nil
			}
		}
		LogError("redis script error cmd:%v errstr:%s", scriptCommitMap[cmd], err)
		return nil, ErrDBErr
	}

	return re, nil
}

type RedisManager struct {
	dbs      []*Redis
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
	r.channels = channels
	r.fun = fun
	for _, v := range r.subMap {
		if v.pubsub != nil {
			v.pubsub.Close()
		}
	}
	for _, v := range r.subMap {
		pubsub := v.Subscribe(channels...)
		v.pubsub = pubsub
		Go(func() {
			for IsRuning() {
				msg, err := pubsub.ReceiveMessage()
				if err == nil {
					fun(msg.Channel, msg.Payload)
				} else if _, ok := err.(net.Error); !ok {
					break
				}
			}
		})
	}
}

func (r *RedisManager) Add(conf *RedisConfig) {
	re := &Redis{
		Client: redis.NewClient(&redis.Options{
			Addr:     conf.Addr,
			Password: conf.Passwd,
			PoolSize: conf.PoolSize,
		}),
		conf:    conf,
		manager: r,
	}
	LogInfo("connect to redis %v", conf.Addr)
	if _, ok := r.subMap[conf.Addr]; !ok {
		r.subMap[conf.Addr] = re
		pubsub := re.Subscribe(r.channels...)
		re.pubsub = pubsub
		Go(func() {
			for IsRuning() {
				msg, err := pubsub.ReceiveMessage()
				if err == nil {
					r.fun(msg.Channel, msg.Payload)
				} else if _, ok := err.(net.Error); !ok {
					break
				}
			}
		})
	}

	r.lock.Lock()
	r.dbs = append(r.dbs, re)
	r.lock.Unlock()
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
	scriptMap             = map[int]string{}
	scriptCommitMap       = map[int]string{}
	scriptHashMap         = map[int]string{}
	scriptIndex     int32 = 0
)

func NewRedisScript(commit, str string) int {
	cmd := int(atomic.AddInt32(&scriptIndex, 1))
	scriptMap[cmd] = str
	scriptCommitMap[cmd] = commit
	return cmd
}

var redisManagers []*RedisManager

func NewRedisManager(conf []*RedisConfig) *RedisManager {
	redisManager := &RedisManager{
		subMap: map[string]*Redis{},
	}
	for _, v := range conf {
		re := &Redis{
			Client: redis.NewClient(&redis.Options{
				Addr:     v.Addr,
				Password: v.Passwd,
				PoolSize: v.PoolSize,
			}),
			conf:    v,
			manager: redisManager,
		}
		LogInfo("connect to redis %v", v.Addr)
		redisManager.subMap[v.Addr] = re
		redisManager.dbs = append(redisManager.dbs, re)
	}

	redisManagers = append(redisManagers, redisManager)
	return redisManager
}
