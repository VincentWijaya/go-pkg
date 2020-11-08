package cache

import (
	"context"
)

type ICache interface {
	Ping() error

	Do(ctx context.Context, command string, args ...interface{}) IReply
	Exists(ctx context.Context, key string) (bool, error)
	TTL(ctx context.Context, key string) IReply

	//Incremental based value
	Incr(ctx context.Context, key string) IReply
	IncrBy(ctx context.Context, key string, incr int) IReply
	Decr(ctx context.Context, key string) IReply
	DecrBy(ctx context.Context, key string, decr int) IReply
	Expire(ctx context.Context, key string, expire int) IReply

	//String based value
	Get(ctx context.Context, key string) IReply
	Set(ctx context.Context, key string, value interface{}) IReply
	SetWithExpire(ctx context.Context, key string, expire int, value interface{}) IReply
	SetNoExpire(ctx context.Context, key string, value interface{}) IReply
	Del(ctx context.Context, key string) IReply
	SetStruct(ctx context.Context, key string, value interface{}) IReply
	SetStructWithExpire(ctx context.Context, key string, expire int, value interface{}) IReply
	SetStructNoExpire(ctx context.Context, key string, value interface{}) IReply

	//Set based value
	SAdd(ctx context.Context, key string, values ...string) IReply
	SRem(ctx context.Context, key string, values ...string) IReply
	SIsMember(ctx context.Context, key, value string) IReply
	SMembers(ctx context.Context, key string) IReply
	SCard(ctx context.Context, key string) IReply

	// Hash based value
	HSet(ctx context.Context, name string, obj interface{}) IReply
	HSetWithExpire(ctx context.Context, name string, expire int, obj interface{}) IReply
	HSetNoExpire(ctx context.Context, name string, obj interface{}) IReply
	HGet(ctx context.Context, name, key string) IReply
	HGetAll(ctx context.Context, name string) IReply
	HDel(ctx context.Context, name string, key string) IReply

	// Sorted Set based value
	ZAdd(ctx context.Context, key string, value interface{}, score int) IReply
	ZRem(ctx context.Context, key string, value interface{}) IReply
	ZRange(ctx context.Context, values ...interface{}) IReply
	ZInterStore(ctx context.Context, values ...interface{}) IReply
	// List based value
}

type IReply interface {
	Error() error
	String() (string, error)
	Float64() (float64, error)
	Int64() (int64, error)
	Int() (int, error)
	Bool() (bool, error)
	Strings() ([]string, error)
	Unmarshal(obj interface{}) error
	Struct(obj interface{}) error
}
