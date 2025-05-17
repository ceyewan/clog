# clog - 灵活强大的Go日志库

`clog`是一个基于[zap](https://github.com/uber-go/zap)和[lumberjack](https://github.com/natefinch/lumberjack)的高性能日志库，提供了多环境支持、结构化日志、多日志器管理和日志文件轮转等功能。

## 特性

- **高性能**：基于uber-go/zap，性能卓越
- **结构化日志**：支持字段化记录，便于分析与检索
- **多环境支持**：为开发、测试和生产环境提供不同配置
- **模块化日志**：为不同模块创建独立日志器，共享同一日志文件
- **智能文件管理**：
  - 自动创建日志目录
  - 文件名中包含时间戳
  - 基于大小、时间和备份数量的日志文件轮转
- **易于使用**：
  - 简洁API设计
  - 支持全局和模块级别日志控制
  - 同时支持结构化和格式化日志记录
- **灵活配置**：支持JSON格式（适用于生产环境）和友好控制台格式（适用于开发）
- **调用者追踪**：可选择性地记录调用位置
- **彩色日志**：控制台输出支持彩色，文件输出自动去除颜色代码

## 安装

```bash
go get -u github.com/ceyewan/clog
```

## 快速开始

```go
package main

import (
    "github.com/ceyewan/clog"
)

func main() {
    // 使用默认配置初始化日志
    config := clog.DefaultConfig()
    config.ConsoleOutput = true // 同时输出到控制台
    
    err := clog.Init(config)
    if err != nil {
        panic(err)
    }
    defer clog.Sync() // 程序结束时确保日志刷新

    // 记录不同级别的日志
    clog.Info("服务启动")
    clog.Warn("警告信息")
    clog.Error("错误信息")
    
    // 结构化日志 - 添加字段
    clog.Info("用户登录", 
        clog.String("username", "admin"), 
        clog.Int("user_id", 12345),
        clog.Bool("is_admin", true),
    )

    // 格式化日志 - 类似Printf
    clog.Infof("处理了 %d 个请求，平均耗时 %.2fms", 100, 12.34)
    
    // 错误日志记录
    err = doSomething()
    if err != nil {
        clog.Error("操作失败", clog.Err(err))
    }
    
    // 创建模块日志器
    userLogger := clog.Module("user")
    orderLogger := clog.Module("order")
    
    // 使用模块日志器
    userLogger.Info("用户模块初始化完成")
    orderLogger.Debug("订单详情", 
        clog.Int("order_id", 10086),
        clog.Float64("amount", 99.99),
    )
}

func doSomething() error {
    // 示例函数
    return nil
}
```

---

## 配置项详解

`clog.Config` 支持如下配置项：

| 字段名         | 类型                | 说明                                      | 默认值              |
|---------------|-------------------|------------------------------------------|-------------------|
| Level         | string            | 日志级别：debug/info/warn/error/fatal      | info             |
| Format        | string            | 日志格式：json/console                      | console          |
| Filename      | string            | 日志文件路径                                | logs/app.log     |
| Name          | string            | 日志器名称（多日志器场景）                     | default          |
| ConsoleOutput | bool              | 是否同时输出到控制台                         | false            |
| EnableCaller  | bool              | 是否记录调用者信息                           | true             |
| EnableColor   | bool              | 控制台输出是否带颜色                         | true             |
| FileRotation  | FileRotationConfig| 文件轮转配置                               | 见下              |

**文件轮转配置（FileRotationConfig）：**

| 字段名      | 类型  | 说明                   | 默认值 |
|------------|------|------------------------|-------|
| MaxSize    | int  | 单文件最大MB (单位：MB)  | 100   |
| MaxBackups | int  | 最多保留文件个数         | 10    |
| MaxAge     | int  | 日志保留天数            | 7     |
| Compress   | bool | 是否压缩轮转文件         | false |

---

## 方法与用法说明

### 初始化与全局操作

- `clog.Init(config)`：初始化全局日志器，建议程序入口调用一次。
- `clog.Sync()`：刷新日志缓冲，建议`defer`在main函数退出前调用。
- `clog.SetEnvironment(env)`：设置全局环境变量（影响文件名等行为）。
- `clog.SetDefaultLevel(level)`：动态调整全局日志级别。

### 日志记录方法

- `clog.Debug/Info/Warn/Error/Panic/Fatal(msg, ...fields)`：结构化日志，支持任意字段。
- `clog.Debugf/Infof/Warnf/Errorf/Panicf/Fatalf(format, ...args)`：格式化日志。
- `clog.With(fields...)`：返回带上下文字段的新日志器。
- `clog.WithFields(map[string]interface{})`：用map批量添加上下文字段。

### 多日志器与模块日志

- `clog.Module(name, ...config)`：为模块创建/获取独立日志器，可单独配置。
- `clog.GetLogger(name)`：获取已注册的日志器。
- `logger.SetLevel(level)`：动态调整某个日志器级别。
- `logger.With(fields...)`：为某个日志器添加上下文字段。
- `logger.Sync()`：刷新该日志器缓冲。
- `logger.Close()`：关闭日志器（如需释放资源）。

### 字段构造器

- `clog.String(key, val)`、`clog.Int(key, val)`、`clog.Bool(key, val)`、`clog.Float64(key, val)`、`clog.Time(key, val)`、`clog.Err(err)` 等，便于结构化日志。

---

## 高级用法与实战示例

### 1. 自定义配置

```go
config := clog.DefaultConfig()
config.Level = clog.DebugLevel
config.Format = clog.FormatJSON
config.Filename = "./logs/myapp.log"
config.ConsoleOutput = true
config.EnableCaller = true
config.EnableColor = true
clog.Init(config)
```

### 2. 环境区分与文件名自动化

```go
clog.SetEnvironment(clog.EnvProduction)
config := clog.DefaultConfig()
config.Format = clog.FormatJSON
config.ConsoleOutput = false
config.EnableColor = false
if clog.GetEnvironment() == clog.EnvDevelopment {
    config.UseTimeStampFilename = true
    config.UsePidFilename = true
}
clog.Init(config)
```

### 3. 多模块日志与独立级别

```go
userLogger := clog.Module("user")
orderLogger := clog.Module("order", clog.Config{
    Level: clog.DebugLevel,
    Filename: "./logs/order.log",
})
userLogger.Info("用户已登录", clog.String("username", "admin"))
orderLogger.Debug("订单调试", clog.Int("order_id", 10086))
orderLogger.SetLevel(clog.ErrorLevel) // 动态调整模块日志级别
```

### 4. 结构化与上下文日志

```go
reqLogger := clog.With(
    clog.String("request_id", "req-123"),
    clog.String("user_agent", "Mozilla/5.0..."),
)
reqLogger.Info("处理请求开始")
reqLogger.Error("处理请求失败", clog.Err(err))
fields := map[string]interface{}{
    "request_id": "req-123",
    "ip": "192.168.1.1",
}
clog.WithFields(fields).Warn("批量字段日志")
```

### 5. 日志轮转与压缩

```go
config := clog.DefaultConfig()
config.FileRotation = &clog.FileRotationConfig{
    MaxSize:    100,    // 单个文件最大尺寸，单位MB
    MaxAge:     7,      // 保留天数
    MaxBackups: 10,     // 最多保留文件个数
    Compress:   true,   // 是否压缩轮转文件
}
clog.Init(config)
```

---

## 常见问题与解答（FAQ）

**Q: 为什么日志文件有颜色代码？**
A: clog 智能区分输出目标，文件输出自动去除颜色，仅控制台可彩色显示。

**Q: 如何只输出到文件/控制台？**
A: 设置 `ConsoleOutput` 为 `false` 只输出到文件，为 `true` 则同时输出到控制台。

**Q: 如何为不同模块设置不同日志级别？**
A: 用 `clog.Module("模块名", clog.Config{Level: clog.DebugLevel})` 创建模块日志器并单独设置级别。

**Q: 如何让日志文件名带时间戳或进程ID？**
A: 在开发环境下设置 `UseTimeStampFilename` 或 `UsePidFilename` 为 `true`。

**Q: 如何保证日志全部写入磁盘？**
A: 程序退出前调用 `clog.Sync()` 或 `clog.SyncAll()`。

**Q: 如何记录结构化错误？**
A: 用 `clog.Err(err)` 字段，便于后续检索和分析。

---

## 最佳实践

1. **生产环境**：
   - 推荐 `FormatJSON`，禁用控制台输出和颜色，开启压缩和轮转。
2. **开发环境**：
   - 推荐 `FormatConsole`，启用颜色和控制台输出，文件名带时间戳/进程ID。
3. **多模块管理**：
   - 每个主要模块用 `clog.Module` 独立日志器，便于分级和定位。
4. **结构化日志**：
   - 用字段而非字符串拼接，关键操作统一字段名，错误日志用 `clog.Err(err)`。
5. **高并发/高性能**：
   - 复用日志器，避免频繁创建临时日志器。
6. **日志刷新**：
   - 程序退出前务必 `defer clog.Sync()` 或 `defer clog.SyncAll()`。

---

## 性能考虑

`clog` 基于高性能的 zap 库，但在以下情况可能影响性能：

- 大量使用 `Debugf`, `Infof` 等格式化函数比直接使用字段开销更大
- 频繁创建临时日志器（如每个请求创建）会有额外开销
- 在高并发场景，建议复用日志器而不是频繁创建

---

## 许可证

[MIT](LICENSE)