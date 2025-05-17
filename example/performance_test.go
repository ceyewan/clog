// 性能测试文件，用于评估clog的性能表现
package main

import (
	"testing"

	"github.com/ceyewan/clog"
)

// BenchmarkBasicLogging 基准测试：基本日志记录性能
func BenchmarkBasicLogging(b *testing.B) {
	// 设置测试配置 - 禁用控制台输出以获取更准确的性能
	config := clog.DefaultConfig()
	config.ConsoleOutput = false
	config.Filename = "logs/perf-basic.log"

	// 初始化日志
	err := clog.Init(config)
	if err != nil {
		b.Fatalf("初始化日志失败: %v", err)
	}
	defer clog.Sync()

	b.ResetTimer()

	// 基本日志记录性能测试
	b.Run("InfoLog", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			clog.Info("这是一条基准测试信息日志")
		}
	})
}

// BenchmarkStructuredLogging 基准测试：结构化日志记录性能
func BenchmarkStructuredLogging(b *testing.B) {
	// 设置测试配置 - 禁用控制台输出以获取更准确的性能
	config := clog.DefaultConfig()
	config.ConsoleOutput = false
	config.Filename = "logs/perf-structured.log"

	// 初始化日志
	err := clog.Init(config)
	if err != nil {
		b.Fatalf("初始化日志失败: %v", err)
	}
	defer clog.Sync()

	b.ResetTimer()

	// 结构化日志记录性能测试
	b.Run("InfoWithFields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			clog.Info("结构化日志基准测试",
				clog.Int("counter", i),
				clog.String("event", "benchmark"),
				clog.Bool("structured", true),
			)
		}
	})
}

// BenchmarkFormatLogging 基准测试：格式化日志记录性能
func BenchmarkFormatLogging(b *testing.B) {
	// 设置测试配置 - 禁用控制台输出以获取更准确的性能
	config := clog.DefaultConfig()
	config.ConsoleOutput = false
	config.Filename = "logs/perf-format.log"

	// 初始化日志
	err := clog.Init(config)
	if err != nil {
		b.Fatalf("初始化日志失败: %v", err)
	}
	defer clog.Sync()

	b.ResetTimer()

	// 格式化日志记录性能测试
	b.Run("Infof", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			clog.Infof("这是第 %d 条格式化日志记录，状态: %s", i, "正在测试")
		}
	})
}

// BenchmarkModuleLogging 基准测试：模块日志记录性能
func BenchmarkModuleLogging(b *testing.B) {
	// 设置测试配置 - 禁用控制台输出以获取更准确的性能
	config := clog.DefaultConfig()
	config.ConsoleOutput = false
	config.Filename = "logs/perf-module.log"

	// 初始化日志
	err := clog.Init(config)
	if err != nil {
		b.Fatalf("初始化日志失败: %v", err)
	}
	defer clog.Sync()

	// 创建测试模块
	moduleLogger := clog.Module("perf-test-module")

	b.ResetTimer()

	// 模块日志记录性能测试
	b.Run("ModuleInfo", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			moduleLogger.Info("模块日志基准测试")
		}
	})

	b.Run("ModuleStructured", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			moduleLogger.Info("模块结构化日志测试",
				clog.Int("iteration", i),
				clog.String("module", "perf-test"),
			)
		}
	})
}
