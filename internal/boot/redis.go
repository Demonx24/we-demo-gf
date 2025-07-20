package boot

import (
	"context"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/database/gredis"
	"log"
)

var RedisClient *gredis.Redis

func InitRedis() {

	client, err := gredis.New()
	if err != nil {
		log.Fatalf("Failed to create Redis client: %v", err)
	}
	RedisClient = client

	ctx := context.Background()
	reply, err := RedisClient.Do(ctx, "PING")
	if err != nil {
		log.Fatalf("Failed to PING Redis: %v", err)
	}
	if reply.String() != "PONG" {
		log.Fatalf("Unexpected PING reply: %s", reply.String())
	}
	log.Println("✅ Redis initialized, PONG received")
}
