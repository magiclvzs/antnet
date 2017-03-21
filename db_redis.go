package antnet

import (
	"net"
	"strings"

	"gopkg.in/redis.v5"
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

func (r *Redis) Script(cmd int, keys []string, args ...interface{}) (interface{}, error) {
	hash, _ := r.manager.script_map_hash[cmd]
	re, err := r.EvalSha(hash, keys, args...).Result()
	if err != nil {
		_, ok := r.manager.script_map[cmd]
		if !ok {
			LogError("redis script error cmd not found cmd:%v", cmd)
			return nil, ErrDBErr
		}

		if strings.HasPrefix(err.Error(), "NOSCRIPT ") {
			LogWarn("try reload redis script")
			for k, v := range r.manager.script_map {
				hash, err := r.ScriptLoad(v).Result()
				if err != nil {
					LogError("redis script load error errstr:%s", err)
					return nil, ErrDBErr
				}
				r.manager.script_map_hash[k] = hash
			}

			hash, _ := r.manager.script_map_hash[cmd]
			re, err := r.EvalSha(hash, keys, args...).Result()
			if err == nil {
				return re, nil
			}
		}
		LogError("redis script error errstr:%s", err)
		return nil, ErrDBErr
	}

	return re, nil
}

type RedisManager struct {
	dbs             []*Redis
	subMap          map[string]*Redis
	script_map      map[int]string
	script_map_hash map[int]string
}

func (r *RedisManager) GetByRid(rid int) *Redis {
	return r.dbs[rid]
}

func (r *RedisManager) GetGlobal() *Redis {
	return r.GetByRid(0)
}

func (r *RedisManager) SetScript(cmd int, str string) {
	if r.script_map == nil {
		r.script_map = map[int]string{}
		r.script_map_hash = map[int]string{}
	}
	r.script_map[cmd] = str
	for _, v := range r.subMap {
		hash, err := v.ScriptLoad(str).Result()
		if err != nil {
			LogError("redis script load error errstr:%s", err)
			break
		}
		r.script_map_hash[cmd] = hash
	}
}

func (r *RedisManager) Sub(fun func(channel, data string), channels ...string) {
	for _, v := range r.subMap {
		if v.pubsub != nil {
			v.pubsub.Close()
		}
	}
	for _, v := range r.subMap {
		pubsub, err := v.Subscribe(channels...)
		if err != nil {
			LogError("redis sub failed:%s", err, pubsub)
		} else {
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
}
func (r *RedisManager) close() {
	for _, v := range r.dbs {
		if v.pubsub != nil {
			v.pubsub.Close()
		}
		v.Close()
	}
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
