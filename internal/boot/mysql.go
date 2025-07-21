package boot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

var MysqlClient *gdb.DB

type MysqlCfg struct {
	Host     string        `json:"host"`
	User     string        `json:"user"`
	Password string        `json:"password"`
	DbName   string        `json:"dbName"`
	Port     int           `json:"port"`
	Charset  string        `json:"charset"`
	Timeout  time.Duration `json:"timeout"`
}

func InitMysql() {
	var mc MysqlCfg
	err := g.Cfg().MustGet(context.Background(), "mysql.default").Struct(&mc)
	if err != nil {
		panic("配置绑定失败：" + err.Error())
	}

	conf := gdb.Config{
		"default": gdb.ConfigGroup{
			{
				Type:    "mysql",
				Host:    mc.Host,
				Port:    fmt.Sprintf("%d", mc.Port),
				User:    mc.User,
				Pass:    mc.Password,
				Name:    mc.DbName,
				Charset: mc.Charset,
			},
		},
	}

	// 先设置配置
	if err := gdb.SetConfig(conf); err != nil {
		panic("设置数据库配置失败：" + err.Error())
	}

	// 再使用无参构造函数创建连接
	client, err := gdb.New()
	if err != nil {
		panic("MySQL 初始化失败：" + err.Error())
	}

	MysqlClient = client

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = MysqlClient.Exec(ctx, "SELECT 1")
	if err != nil {
		panic("MySQL ping 失败：" + err.Error())
	}

	log.Println("MySQL 初始化成功")
}
