package cache

import (
	"Clould/webook/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrKeyNotExist = redis.Nil

type Cache interface {
	Get(ctx context.Context, id int64) (domain.User, error)
	// 读取文章
	GetAircle(ctx context.Context, id int64) (domain.User, error)
	// 其他的业务
}

type CacheV1 interface {
	Get(ctx context.Context, key string) (any, error)
}

type UserCache struct {
	// cache CacheV1 中间件做法
	// 面向接口编程 传单机redis 可以
	client redis.Cmdable
	// 过期时间
	expiration time.Duration
}

// A  用到了 B  ，B是接口， B是A的字段， A不是初始化B， 而是从外面注入
func NewUserCache(client redis.Cmdable) *UserCache {
	return &UserCache{
		client:     client,
		expiration: time.Minute * 15,
	}
}

func (cache *UserCache) key(id int64) string {
	// bumen_xiaozuz_user_info_key
	return fmt.Sprintf("user:info:%d", id)
}

// 只要 error为nil 认为缓存有数据
// 如果没有数据就返回一个特定的error
func (cache *UserCache) Get(ctx context.Context, id int64) (domain.User, error) {
	key := cache.key(id)
	// 数据不存在， err = redis.NiL
	val, err := cache.client.Get(ctx, key).Bytes()
	if err != nil {
		return domain.User{}, err
	}

	var u domain.User
	err = json.Unmarshal(val, &u)
	if err != nil {
		return domain.User{}, err
	}
	return u, err
}

func (cache *UserCache) Set(ctx context.Context, u domain.User) error {
	val, err := json.Marshal(u)
	if err != nil {
		return err
	}
	key := cache.key(u.Id)
	return cache.client.Set(ctx, key, val, cache.expiration).Err()
}

func (cache *UserCache) GetUser(ctx context.Context, id int64) (user domain.User, err error) {
	return domain.User{}, err
}
