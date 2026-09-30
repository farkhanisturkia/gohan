package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/farkhanisturkia/gohan/pkg/internal/database"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/singleflight"
)

var inflight singleflight.Group

func CachedData(
	ctx context.Context,
	rdb *redis.Client,
	key string,
	ttl time.Duration,
	dest interface{},
	loader func() (interface{}, error),
) (interface{}, bool, error) {
	if rdb != nil {
		cachedData, err := rdb.Get(ctx, key).Result()
		if err == nil && cachedData != "" {
			if uerr := json.Unmarshal([]byte(cachedData), dest); uerr == nil {
				return dest, true, nil
			}
		}
	}

	val, err, _ := inflight.Do(key, func() (interface{}, error) {
		data, lerr := loader()
		if lerr != nil {
			return nil, lerr
		}

		if rdb != nil {
			if jsonBytes, merr := json.Marshal(data); merr == nil {
				_ = rdb.Set(ctx, key, string(jsonBytes), ttl).Err()
			}
		}

		return data, nil
	})
	if err != nil {
		return nil, false, err
	}

	return val, false, nil
}

func CacheForget(ctx context.Context, rdb *redis.Client, keys ...string) {
	for _, key := range keys {
		inflight.Forget(key)
		if rdb != nil {
			_ = rdb.Del(ctx, key).Err()
		}
	}
}

func IsNotFound(err error) bool {
	return err == database.ErrRecordNotFound
}
