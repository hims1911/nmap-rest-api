package databse

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis(ctx context.Context) {
	RDB = redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "", // no auth
		DB:       0,
	})
	if err := RDB.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("redis has been connected")
}
