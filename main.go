package main

import (
	"context"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"
	"github.com/gogf/gf/v2/os/gctx"
	"we-demo-gf/internal/boot"
	_ "we-demo-gf/internal/packed"

	"we-demo-gf/internal/cmd"
)

func main() {
	boot.InitMQ()
	boot.StartConsumer(context.Background())
	//boot.InitRedis()
	defer boot.RedisClient.Close(context.Background())
	cmd.Main.Run(gctx.GetInitCtx())
}
