# Clog - 高性能结构化日志库

Clog 是一个基于 [uber-go/zap](https://github.com/uber-go/zap) 构建的高性能、灵活的结构化日志库，专为 Go 应用程序设计。它提供了简洁的 API，同时支持结构化日志和人类友好的输出格式。

## 核心特性

- **高性能**：基于 zap 的高性能日志引擎，专为生产环境优化
- **灵活配置**：支持 JSON 和控制台格式，可根据环境需求灵活切换
- **日志级别**：支持 debug、info、warn、error、panic、fatal 多种日志级别
- **日志轮转**：内置基于 [lumberjack](https://github.com/natefinch/lumberjack) 的日志文件轮转功能
- **结构化日志**：支持字段化日志记录，便于日志分析和处理
- **彩色输出**：开发环境下支持彩色日志输出，提高可读性

## 实现原理

Clog 基于以下原则设计：

1. **封装复杂性**：对 zap 的底层 API 进行封装，提供更简洁易用的接口
2. **灵活性与性能平衡**：在保持高性能的同时，提供足够的灵活性
3. **多环境适应**：同时适应开发环境和生产环境的需求

核心组件：

- **Logger**：主要的日志记录器，封装了 zap 的功能
- **Config**：日志配置，控制日志的行为和输出

## 安装

```bash
go get github.com/yourusername/clog
```

## 基本用法

### 初始化默认日志器

```go
package main

import (
    "github.com/yourusername/clog"
)

func main() {
    // 使用默认配置初始化日志器
    err := clog.Init(clog.DefaultConfig())
    if err != nil {
        panic(err)
    }
    
    // 程序结束前确保日志被刷新
    defer clog.Sync()
    
    // 记录日志
    clog.Info("应用启动成功")
    clog.Warn("这是一条警告信息")
    clog.Error("发生错误", clog.String("reason", "配置无效"))
}
```

### 自定义配置

```go
config := clog.Config{
    Level:         clog.InfoLevel,
    Format:        clog.FormatJSON,      // 使用JSON格式，适合生产环境
    Filename:      "./logs/app.log",
    ConsoleOutput: true,                 // 同时输出到控制台
    EnableCaller:  true,                 // 记录调用者信息
    EnableColor:   false,                // 在JSON模式下颜色无效
    FileRotation: &clog.FileRotationConfig{
        MaxSize:    100,                 // 单个文件最大100MB
        MaxAge:     7,                   // 保留7天
        MaxBackups: 10,                  // 最多保留10个备份
        Compress:   true,                // 压缩旧文件
    },
}

err := clog.Init(config)
```

### 结构化日志

```go
// 使用预定义的字段构造函数
clog.Info("用户登录", 
    clog.String("username", "admin"),
    clog.Int("user_id", 10001),
    clog.Bool("is_admin", true),
)

// 添加错误信息
err := doSomething()
if err != nil {
    clog.Error("操作失败", clog.Err(err))
}

// 使用WithFields添加固定上下文
logger := clog.WithFields(map[string]interface{}{
    "module": "user_service",
    "version": "1.0.0",
})
logger.Info("服务初始化")
```

### 动态调整日志级别

```go
// 全局调整默认日志器的级别
clog.SetDefaultLevel(clog.DebugLevel)

// 对特定日志器实例调整级别
logger := clog.NewLogger(config)
logger.SetLevel(clog.WarnLevel)
```

## 性能考虑

- 对于性能敏感的应用，建议在生产环境使用 JSON 格式
- 使用字段化日志而非格式化字符串（如 `Info()` 而非 `Infof()`）可获得更好的性能
- 避免在热路径中创建临时 logger 实例

## 扩展

Clog 设计为可扩展的，您可以：

- 使用 `GetZapLogger()` 获取底层 zap.Logger 实例
- 创建自定义的日志适配器扩展功能
- 实现自己的输出格式和目标

## 许可证

MIT

## 致谢

Clog 基于以下出色的开源项目：

- [uber-go/zap](https://github.com/uber-go/zap) - 高性能日志库
- [natefinch/lumberjack](https://github.com/natefinch/lumberjack) - 日志轮转功能
