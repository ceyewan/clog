# clog - 简单易用的 Go 日志库

clog 是一个轻量级的 Go 日志库，提供简单易用的 API 接口，满足基本的日志记录需求。

## 特性

- 多级别日志支持：提供 Debug、Info、Warning、Error 四个日志级别
- 灵活的输出目标：支持同时输出到控制台和文件
- 彩色日志输出：控制台输出时自动为不同级别的日志添加颜色标识
- 自动日志分割：按日期自动分割日志文件
- 线程安全：内部使用互斥锁保证并发安全
- 源代码信息：自动记录调用者的文件名和行号
- 时间戳：每条日志自动添加时间戳

## 使用示例

```go
package main

import (
    "github.com/yourusername/clog"
)

func main() {
    // 设置日志输出路径，如果为空则只输出到控制台
    clog.SetLogPath("./logs")
    
    // 设置日志级别，只有大于等于该级别的日志会被记录
    clog.SetLogLevel(clog.LevelDebug)
    
    // 记录不同级别的日志
    clog.Debug("这是一条 %s 日志", "调试")
    clog.Info("这是一条 %s 日志", "信息")
    clog.Warning("这是一条 %s 日志", "警告")
    clog.Error("这是一条 %s 日志", "错误")
    
    // 程序结束前关闭日志系统
    defer clog.Close()
}
```

## API 说明

- Debug(format string, args ...interface{})：记录调试级别日志
- Info(format string, args ...interface{})：记录信息级别日志
- Warning(format string, args ...interface{})：记录警告级别日志
- Error(format string, args ...interface{})：记录错误级别日志
- SetLogPath(logDir string) error：设置日志文件输出目录
- SetLogLevel(level LogLevel)：设置日志记录级别
- Close() error：关闭日志系统，释放资源

## 日志级别

```go
const (
    LevelDebug   // 调试级别，最详细的日志信息
    LevelInfo    // 信息级别，常规日志信息
    LevelWarning // 警告级别，需要注意的问题
    LevelError   // 错误级别，程序错误信息
)
```

## 日志输出

```text
[级别] 时间戳 文件名:行号 日志内容
```
