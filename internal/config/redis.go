package config

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func InitRedis(ctx context.Context) (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	password := os.Getenv("REDIS_PASSWORD")
	db := 0
	if os.Getenv("REDIS_DB") != "" {
		fmt.Sscanf(os.Getenv("REDIS_DB"), "%d", &db)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
