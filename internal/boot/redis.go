package boot

import (
	"context"
	"log"
	"time"

	"github.com/gogf/gf/v2/database/gredis" // 使用 gf 内置的 gredis
	"github.com/gogf/gf/v2/frame/g"
)

var RedisClient *gredis.Redis

type RedisCfg struct {
	Address     string        `json:"address"`
	Pass        string        `json:"pass"`
	Db          int           `json:"db"`
	IdleTimeout time.Duration `json:"idleTimeout"`
	MaxActive   int           `json:"maxActive"`
	MaxIdle     int           `json:"maxIdle"`
	MinIdle     int           `json:"minIdle"`
}

func InitRedis() {
	// 读取配置
	var rc RedisCfg
	err := g.Cfg().MustGet(context.Background(), "redis.default").Struct(&rc)
	if err != nil {
		panic("配置绑定失败：" + err.Error())
	}

	// 创建 gredis.Config 配置
	conf := &gredis.Config{
		Address:     rc.Address,
		Pass:        rc.Pass,
		Db:          rc.Db,
		IdleTimeout: rc.IdleTimeout,
		MaxActive:   rc.MaxActive,
		MaxIdle:     rc.MaxIdle,
		MinIdle:     rc.MinIdle,
	}

	// 使用 gredis.New 创建 Redis 客户端实例
	client, err := gredis.New(conf)
	if err != nil {
		panic("Redis 初始化失败：" + err.Error())
	}

	RedisClient = client

	// 测试 Redis 是否正常
	ctxPing, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	reply, err := RedisClient.Do(ctxPing, "PING")
	if err != nil || reply.String() != "PONG" {
		panic("Redis ping failed: " + err.Error())
	}

	log.Println(" Redis 初始化成功")
}
