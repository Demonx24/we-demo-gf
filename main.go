package main

import (
	"context"
	_ "github.com/gogf/gf/contrib/drivers/mysql/v2" // 导入 MySQL 驱动
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"   // 导入 Redis 驱动
	"github.com/gogf/gf/v2/os/gctx"
	"we-demo-gf/internal/boot"
	"we-demo-gf/internal/cmd"
	_ "we-demo-gf/internal/packed"
)

func main() {

	// 初始化 Redis 和 MQ 等
	boot.InitMysql()
	boot.InitRedis()
	boot.InitMQ()
	boot.StartConsumer(context.Background())
	defer boot.RedisClient.Close(context.Background())

	// 启动命令行
	cmd.Main.Run(gctx.GetInitCtx())
}
