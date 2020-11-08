package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

//-------------------
type RedisConfig struct {
	Connection string
	Password   string
	Timeout    int
	MaxIdle    int
	MaxActive  int
}

type Redis struct {
	connection string
	timeout    time.Duration
	pool       *redis.Pool
}

type Reply struct {
	result interface{}
	error  error
}

const ErrorFailedConnect = "Failed to connect to redis %s. Error: %s"

// ErrorNil redis error no data
var ErrorNil = redis.ErrNil

func ConnectRedis(config RedisConfig) (ICache, error) {
	timeout := time.Duration(config.Timeout) * time.Second
	pool := &redis.Pool{
		MaxIdle:     config.MaxIdle,
		MaxActive:   config.MaxActive,
		IdleTimeout: timeout,
		Wait:        true,
		Dial: func() (redis.Conn, error) {
			conn, err := redis.Dial("tcp", config.Connection, redis.DialConnectTimeout(timeout))
			if err != nil {
				return nil, err
			}
			return conn, err
		},
	}

	conn, _ := pool.Get().(redis.ConnWithTimeout)
	_, err := conn.DoWithTimeout(timeout, "PING")
	if err != nil {
		return nil, fmt.Errorf(ErrorFailedConnect, config.Connection, err)
	}

	return &Redis{connection: config.Connection, timeout: timeout, pool: pool}, nil
}

func (r *Redis) getConnection() redis.ConnWithTimeout {
	return r.pool.Get().(redis.ConnWithTimeout)
}

func (r *Redis) Do(ctx context.Context, command string, args ...interface{}) IReply {
	conn := r.getConnection()
	defer conn.Close()

	result, err := conn.DoWithTimeout(r.timeout, command, args...)
	return &Reply{result: result, error: err}
}

func (r *Redis) Ping() error {
	reply, err := r.Do(context.Background(), "PING").String()
	if err != nil || reply != "PONG" {
		return fmt.Errorf(ErrorFailedConnect, r.connection, err)
	}
	return nil
}

func (r *Redis) Exists(ctx context.Context, key string) (bool, error) {
	reply, err := r.Do(ctx, "EXISTS", key).Int()
	if err != nil {
		return false, fmt.Errorf(ErrorFailedConnect, r.connection, err)
	}

	return reply == 1, nil
}
func (r *Redis) TTL(ctx context.Context, key string) IReply {
	return r.Do(ctx, "TTL", key)
}
func (r *Redis) Expire(ctx context.Context, key string, expire int) IReply {
	return r.Do(ctx, "EXPIRE", key, expire)
}
func (r *Redis) Incr(ctx context.Context, key string) IReply {
	return r.Do(ctx, "INCR", key)
}
func (r *Redis) IncrBy(ctx context.Context, key string, incr int) IReply {
	return r.Do(ctx, "INCRBY", key, incr)
}
func (r *Redis) Decr(ctx context.Context, key string) IReply {
	return r.Do(ctx, "DECR", key)
}
func (r *Redis) DecrBy(ctx context.Context, key string, decr int) IReply {
	return r.Do(ctx, "DECRBY", key, decr)
}
func (r *Redis) Get(ctx context.Context, key string) IReply {
	return r.Do(ctx, "GET", key)
}
func (r *Redis) Set(ctx context.Context, key string, value interface{}) IReply {
	result := r.Do(ctx, "SET", key, value)
	r.Expire(ctx, key, 15*60)
	return result
}
func (r *Redis) SetWithExpire(ctx context.Context, key string, expire int, value interface{}) IReply {
	result := r.Do(ctx, "SET", key, value)
	r.Expire(ctx, key, expire)
	return result
}
func (r *Redis) SetNoExpire(ctx context.Context, key string, value interface{}) IReply {
	return r.Do(ctx, "SET", key, value)
}
func (r *Redis) Del(ctx context.Context, key string) IReply {
	return r.Do(ctx, "DEL", key)
}
func (r *Redis) SetStruct(ctx context.Context, key string, value interface{}) IReply {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return &Reply{result: nil, error: err}
	}
	return r.Set(ctx, key, jsonValue)
}
func (r *Redis) SetStructWithExpire(ctx context.Context, key string, expire int, value interface{}) IReply {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return &Reply{result: nil, error: err}
	}
	return r.SetWithExpire(ctx, key, expire, jsonValue)
}
func (r *Redis) SetStructNoExpire(ctx context.Context, key string, value interface{}) IReply {
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return &Reply{result: nil, error: err}
	}
	return r.SetNoExpire(ctx, key, jsonValue)
}
func (r *Redis) SAdd(ctx context.Context, key string, values ...string) IReply {
	args := stringToInterface(key, values...)
	result := r.Do(ctx, "SADD", args...)
	r.Expire(ctx, key, 15*60)
	return result
}
func (r *Redis) SAddWithExpire(ctx context.Context, key string, expire int, values ...string) IReply {
	args := stringToInterface(key, values...)
	return r.Do(ctx, "SADD", args...)
}
func (r *Redis) SAddNoExpire(ctx context.Context, key string, values ...string) IReply {
	args := stringToInterface(key, values...)
	return r.Do(ctx, "SADD", args...)
}
func (r *Redis) SRem(ctx context.Context, key string, values ...string) IReply {
	args := stringToInterface(key, values...)
	return r.Do(ctx, "SREM", args...)
}
func (r *Redis) SIsMember(ctx context.Context, key, value string) IReply {
	return r.Do(ctx, "SISMEMBER", key, value)
}
func (r *Redis) SMembers(ctx context.Context, key string) IReply {
	return r.Do(ctx, "SMEMBERS", key)
}
func (r *Redis) SCard(ctx context.Context, key string) IReply {
	return r.Do(ctx, "SCARD", key)
}
func (r *Redis) HSet(ctx context.Context, name string, obj interface{}) IReply {
	result := r.Do(ctx, "HMSET", redis.Args{}.Add(name).AddFlat(obj)...)
	r.Expire(ctx, name, 15*60)
	return result
}
func (r *Redis) HSetWithExpire(ctx context.Context, name string, expire int, obj interface{}) IReply {
	result := r.Do(ctx, "HMSET", redis.Args{}.Add(name).AddFlat(obj)...)
	r.Expire(ctx, name, expire)
	return result
}
func (r *Redis) HSetNoExpire(ctx context.Context, name string, obj interface{}) IReply {
	return r.Do(ctx, "HMSET", redis.Args{}.Add(name).AddFlat(obj)...)
}
func (r *Redis) HGet(ctx context.Context, name, key string) IReply {
	return r.Do(ctx, "HGET", redis.Args{}.Add(name).Add(key)...)
}
func (r *Redis) HGetAll(ctx context.Context, name string) IReply {
	return r.Do(ctx, "HGETALL", name)
}
func (r *Redis) HDel(ctx context.Context, name, key string) IReply {
	return r.Do(ctx, "HDEL", redis.Args{}.Add(name).Add(key)...)
}

func (r *Redis) ZAdd(ctx context.Context, key string, value interface{}, score int) IReply {
	return r.Do(ctx, "ZADD", key, score, value)
}

func (r *Redis) ZRem(ctx context.Context, key string, value interface{}) IReply {
	return r.Do(ctx, "ZREM", key, value)
}

func (r *Redis) ZRange(ctx context.Context, values ...interface{}) IReply {
	return r.Do(ctx, "ZRANGE", values...)
}

func (r *Redis) ZInterStore(ctx context.Context, values ...interface{}) IReply {
	return r.Do(ctx, "ZINTERSTORE", values...)
}

func (rp *Reply) Unmarshal(obj interface{}) error {
	b, err := redis.Bytes(rp.result, rp.error)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}
	return nil
}
func (rp *Reply) Error() error {
	return rp.error
}
func (rp *Reply) String() (string, error) {
	return redis.String(rp.result, rp.error)
}
func (rp *Reply) Float64() (float64, error) {
	return redis.Float64(rp.result, rp.error)
}
func (rp *Reply) Int64() (int64, error) {
	return redis.Int64(rp.result, rp.error)
}
func (rp *Reply) Int() (int, error) {
	return redis.Int(rp.result, rp.error)
}
func (rp *Reply) Bool() (bool, error) {
	return redis.Bool(rp.result, rp.error)
}
func (rp *Reply) Strings() ([]string, error) {
	return redis.Strings(rp.result, rp.error)
}
func (rp *Reply) Bytes() ([]byte, error) {
	return redis.Bytes(rp.result, rp.error)
}
func (rp *Reply) Struct(obj interface{}) error {
	result, err := redis.Values(rp.result, rp.error)
	if err != nil {
		return err
	}
	if err = redis.ScanStruct(result, obj); err != nil {
		return err
	}
	return nil
}

func stringToInterface(key string, values ...string) []interface{} {
	var args []interface{}
	args = append(args, key)
	for _, v := range values {
		args = append(args, v)
	}
	return args
}
