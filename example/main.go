package main

import (
	"errors"
	"time"

	"github.com/ceyewan/clog"
)

func main() {
	// 使用默认配置初始化（开发环境）
	config := clog.DefaultConfig()
	config.ConsoleOutput = true // 输出到控制台

	err := clog.Init(config)
	if err != nil {
		panic(err)
	}

	// 程序结束时确保所有日志都被写入
	defer clog.Sync()

	// 基本日志记录
	clog.Info("程序启动")
	clog.Debug("这是一条调试信息")
	clog.Warn("这是一条警告信息")

	// 结构化日志记录
	clog.Info("用户登录",
		clog.String("username", "admin"),
		clog.Int("user_id", 1001),
		clog.Bool("is_admin", true),
		clog.Time("login_time", time.Now()),
	)

	// 格式化日志
	clog.Infof("处理了 %d 个请求，耗时 %.2f 秒", 100, 0.125)

	// 错误日志
	err = errors.New("数据库连接失败")
	clog.Error("操作失败", clog.Err(err))

	// 为不同模块创建日志器
	userLogger := clog.Module("user")
	orderLogger := clog.Module("order", clog.Config{
		Level: clog.DebugLevel,
	})

	// 使用模块日志器
	userLogger.Info("用户模块初始化")
	orderLogger.Debug("订单模块调试信息")
	orderLogger.Info("创建订单",
		clog.Int("order_id", 10086),
		clog.Float64("amount", 199.99),
	)

	// 动态修改日志级别
	clog.Info("当前日志级别为 INFO")
	clog.Debug("这条调试日志可以看到") // 因为默认级别是INFO，所以这条日志不会显示

	clog.SetDefaultLevel(clog.DebugLevel)
	clog.Info("日志级别已改为 DEBUG")
	clog.Debug("这条调试日志现在可以看到了")

	clog.Info("程序运行完毕")
}
