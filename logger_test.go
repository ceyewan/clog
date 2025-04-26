package clog

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

// 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, InfoLevel, cfg.Level)
	assert.Equal(t, FormatConsole, cfg.Format)
	assert.Equal(t, "./logs/app.log", cfg.Filename)
	assert.True(t, cfg.ConsoleOutput)
	assert.True(t, cfg.EnableCaller)
	assert.True(t, cfg.EnableColor)

	assert.NotNil(t, cfg.FileRotation)
	assert.Equal(t, 100, cfg.FileRotation.MaxSize)
	assert.Equal(t, 7, cfg.FileRotation.MaxAge)
	assert.Equal(t, 10, cfg.FileRotation.MaxBackups)
	assert.False(t, cfg.FileRotation.Compress)
}

// 测试日志级别解析
func TestParseLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected zapcore.Level
	}{
		{DebugLevel, zapcore.DebugLevel},
		{InfoLevel, zapcore.InfoLevel},
		{WarnLevel, zapcore.WarnLevel},
		{ErrorLevel, zapcore.ErrorLevel},
		{PanicLevel, zapcore.PanicLevel},
		{FatalLevel, zapcore.FatalLevel},
		{"unknown", zapcore.InfoLevel}, // 默认为 info
	}

	for _, test := range tests {
		t.Run(test.level, func(t *testing.T) {
			result := parseLevel(test.level)
			assert.Equal(t, test.expected, result)
		})
	}
}

// 测试创建新的日志器
func TestNewLogger(t *testing.T) {
	// 创建临时日志目录
	tmpDir, err := os.MkdirTemp("", "clog-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	logFile := filepath.Join(tmpDir, "test.log")

	config := Config{
		Level:         DebugLevel,
		Format:        FormatJSON,
		Filename:      logFile,
		ConsoleOutput: false,
		EnableCaller:  true,
		FileRotation: &FileRotationConfig{
			MaxSize:    1,
			MaxAge:     1,
			MaxBackups: 1,
			Compress:   false,
		},
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)
	assert.NotNil(t, logger)

	logger.Info("test message", String("key", "value"))
	logger.Sync()

	// 验证日志文件是否创建
	_, err = os.Stat(logFile)
	assert.NoError(t, err)
}

// 测试设置日志级别
func TestSetLevel(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clog-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := Config{
		Level:         InfoLevel,
		Format:        FormatConsole,
		Filename:      filepath.Join(tmpDir, "level-test.log"),
		ConsoleOutput: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// 初始为 info 级别，debug 不应该记录
	logger.Debug("this should not be logged")

	// 设置为 debug 级别
	logger.SetLevel(DebugLevel)
	logger.Debug("this should be logged")

	logger.Sync()
}

// 测试 With 字段
func TestWithFields(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clog-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := Config{
		Level:         DebugLevel,
		Format:        FormatJSON,
		Filename:      filepath.Join(tmpDir, "fields-test.log"),
		ConsoleOutput: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// 创建带字段的日志器
	fieldLogger := logger.With(String("service", "test-service"))
	assert.NotNil(t, fieldLogger)

	// 使用 map 创建带字段的日志器
	mapLogger := logger.WithFields(map[string]interface{}{
		"component": "test-component",
		"version":   1.0,
	})
	assert.NotNil(t, mapLogger)

	// 记录日志
	fieldLogger.Info("field logger test")
	mapLogger.Info("map logger test")

	logger.Sync()
}

// 测试初始化默认日志器
func TestInit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clog-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := Config{
		Level:         InfoLevel,
		Format:        FormatConsole,
		Filename:      filepath.Join(tmpDir, "default-test.log"),
		ConsoleOutput: false,
	}

	err = Init(config)
	require.NoError(t, err)

	// 使用全局函数记录日志
	Info("global info message")
	Warn("global warn message", String("test", "value"))

	// 使用全局With函数
	logger := With(String("global", "field"))
	require.NotNil(t, logger)
	logger.Info("with global field")

	Sync()
}

// 测试各种日志级别的方法
func TestLogMethods(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "clog-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := Config{
		Level:         DebugLevel,
		Format:        FormatJSON,
		Filename:      filepath.Join(tmpDir, "methods-test.log"),
		ConsoleOutput: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// 测试基本日志方法
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	// 测试带字段的方法
	logger.Debug("debug with fields", String("level", "debug"))
	logger.Info("info with fields", Int("count", 5))
	logger.Warn("warn with fields", Bool("important", true))
	logger.Error("error with fields", Err(fmt.Errorf("test error")))

	// 测试格式化方法
	logger.Debugf("formatted %s", "debug")
	logger.Infof("formatted %s", "info")
	logger.Warnf("formatted %s", "warn")
	logger.Errorf("formatted %s", "error")

	// 测试字段辅助函数
	logger.Info("field helpers",
		String("string", "value"),
		Int("int", 123),
		Int64("int64", int64(123)),
		Float64("float64", 123.456),
		Bool("bool", true),
		Time("time", time.Now()),
		Duration("duration", time.Second),
		Any("any", struct{ Name string }{"test"}),
	)

	logger.Sync()
}

// 确保 Panic/Fatal 不在正常测试中运行
func TestPanicAndFatalAreSafe(t *testing.T) {
	if os.Getenv("ACTUALLY_TEST_PANIC_FATAL") != "1" {
		t.Skip("Skipping panic/fatal tests. Set ACTUALLY_TEST_PANIC_FATAL=1 to enable.")
		return
	}

	tmpDir, err := os.MkdirTemp("", "clog-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := Config{
		Level:         DebugLevel,
		Format:        FormatJSON,
		Filename:      filepath.Join(tmpDir, "panic-test.log"),
		ConsoleOutput: false,
	}

	logger, err := NewLogger(config)
	require.NoError(t, err)

	// 这些方法会导致程序崩溃或退出，所以在正常测试中被跳过
	logger.Panic("panic message")
	logger.Panicf("panic formatted %s", "message")
	logger.Fatal("fatal message")
	logger.Fatalf("fatal formatted %s", "message")
}
