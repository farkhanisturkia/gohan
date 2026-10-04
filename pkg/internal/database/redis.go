package database

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func InitRedis(host, port, password string) {
	addr := fmt.Sprintf("%s:%s", host, port)
	if host == "" {
		addr = "localhost:6379"
	}

	Redis = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx := context.Background()
	_, err := Redis.Ping(ctx).Result()
	if err != nil {
		log.Printf("[error] Failed to connect to Redis: %v. Caching will be disabled.\n", err)
	} else {
		log.Println("[info] Connect to the Redis server at", addr)
	}
}

func BindRedis(slot **redis.Client) func(host, port, password string) {
	return func(host, port, password string) {
		InitRedis(host, port, password)
		*slot = Redis
	}
}
