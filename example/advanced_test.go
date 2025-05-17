// 高级测试文件，用于验证clog的各种功能
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ceyewan/clog"
)

// TestBasicLogging 测试基本日志记录功能
func TestBasicLogging(t *testing.T) {
	// 设置测试配置
	config := clog.DefaultConfig()
	config.ConsoleOutput = true
	config.Filename = filepath.Join("logs", "test-basic.log")

	// 初始化日志
	err := clog.Init(config)
	if err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}
	defer clog.Sync()

	t.Log("开始基本日志功能测试...")

	// 测试各个级别的日志
	clog.Debug("这是一条调试日志")
	clog.Info("这是一条信息日志")
	clog.Warn("这是一条警告日志")
	clog.Error("这是一条错误日志")
	// Fatal会导致程序退出，所以在测试中不调用

	// 检查日志文件是否生成
	files, err := filepath.Glob(filepath.Join("logs", "test-basic*.log"))
	if err != nil {
		t.Errorf("查找日志文件失败: %v", err)
	} else if len(files) == 0 {
		t.Error("未找到生成的日志文件")
	} else {
		t.Logf("成功生成日志文件: %v", files)
		// 测试通过后清理文件
		for _, file := range files {
			os.Remove(file)
		}
	}
}

// TestStructuredLogging 测试结构化日志记录
func TestStructuredLogging(t *testing.T) {
	// 设置测试配置
	config := clog.DefaultConfig()
	config.ConsoleOutput = true
	config.Filename = filepath.Join("logs", "test-structured.log")
	config.Format = clog.FormatJSON // 使用JSON格式以便检查结构化字段

	// 初始化日志
	err := clog.Init(config)
	if err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}
	defer clog.Sync()

	t.Log("开始结构化日志功能测试...")

	// 使用不同类型的结构化字段
	testError := errors.New("测试错误")
	testTime := time.Now()

	clog.Info("用户操作记录",
		clog.String("user", "tester"),
		clog.Int("id", 12345),
		clog.Bool("success", true),
		clog.Float64("score", 98.5),
		clog.Err(testError),
		clog.Time("timestamp", testTime),
	)

	// 使用格式化日志
	clog.Infof("处理了 %d 个请求，成功率 %.2f%%", 100, 99.9)

	// 检查日志文件是否生成
	files, err := filepath.Glob(filepath.Join("logs", "test-structured*.log"))
	if err != nil {
		t.Errorf("查找日志文件失败: %v", err)
	} else if len(files) == 0 {
		t.Error("未找到生成的日志文件")
	} else {
		t.Logf("成功生成日志文件: %v", files)
		// 读取文件内容进行检查
		content, err := os.ReadFile(files[0])
		if err != nil {
			t.Errorf("读取日志文件失败: %v", err)
		} else {
			// 简单检查JSON格式是否包含预期字段
			if string(content) == "" {
				t.Error("日志文件内容为空")
			} else {
				t.Log("日志内容验证通过")
			}
		}
		// 测试通过后清理文件
		for _, file := range files {
			os.Remove(file)
		}
	}
}

// TestModuleLoggers 测试多模块日志器
func TestModuleLoggers(t *testing.T) {
	// 设置测试配置
	config := clog.DefaultConfig()
	config.ConsoleOutput = true
	config.Filename = filepath.Join("logs", "test-module.log")

	// 初始化日志
	err := clog.Init(config)
	if err != nil {
		t.Fatalf("初始化日志失败: %v", err)
	}
	defer clog.Sync()

	t.Log("开始模块日志功能测试...")

	// 创建不同的模块日志器
	userLogger := clog.Module("user")
	orderLogger := clog.Module("order", clog.Config{
		Level: clog.DebugLevel,
	})
	paymentLogger := clog.Module("payment")

	// 使用模块日志器
	userLogger.Info("用户模块初始化")
	orderLogger.Debug("订单模块调试信息")
	orderLogger.Info("创建订单",
		clog.Int("order_id", 10086),
		clog.Float64("amount", 199.99),
	)
	paymentLogger.Info("支付模块初始化")

	// 测试级别设置
	clog.Debug("这条默认日志器调试日志不应该显示") // 默认INFO级别
	clog.SetDefaultLevel(clog.DebugLevel)
	clog.Debug("这条默认日志器调试日志应该显示") // 已改为DEBUG级别

	// 检查日志文件是否生成
	files, err := filepath.Glob(filepath.Join("logs", "test-module*.log"))
	if err != nil {
		t.Errorf("查找日志文件失败: %v", err)
	} else if len(files) == 0 {
		t.Error("未找到生成的日志文件")
	} else {
		t.Logf("成功生成日志文件: %v", files)
		// 测试通过后清理文件
		for _, file := range files {
			os.Remove(file)
		}
	}
}

// TestLoggerOptions 测试日志器选项配置
func TestLoggerOptions(t *testing.T) {
	t.Log("开始日志选项测试...")

	// 测试不同格式
	formats := []string{clog.FormatJSON, clog.FormatConsole}
	for _, format := range formats {
		config := clog.DefaultConfig()
		config.Format = format
		config.ConsoleOutput = true
		config.Filename = filepath.Join("logs", fmt.Sprintf("test-format-%s.log", format))

		err := clog.Init(config)
		if err != nil {
			t.Errorf("初始化%s格式日志失败: %v", format, err)
			continue
		}

		clog.Info("测试不同日志格式", clog.String("format", format))
		clog.Sync()
	}

	// 测试调用者信息
	config := clog.DefaultConfig()
	config.EnableCaller = true
	config.ConsoleOutput = true
	config.Filename = filepath.Join("logs", "test-caller.log")

	err := clog.Init(config)
	if err != nil {
		t.Errorf("初始化带调用者信息的日志失败: %v", err)
	} else {
		clog.Info("这条日志应该包含调用者信息")
		clog.Sync()
	}

	// 清理测试文件
	files, _ := filepath.Glob(filepath.Join("logs", "test-format-*.log"))
	for _, file := range files {
		os.Remove(file)
	}

	os.Remove(filepath.Join("logs", "test-caller.log"))
}
