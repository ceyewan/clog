// Package clog 提供一个灵活的日志系统，基于 uber-go/zap
// 支持结构化日志和人类友好的输出格式
package clog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 定义日志级别常量
const (
	// DebugLevel 表示调试级别日志
	DebugLevel = "debug"
	// InfoLevel 表示信息级别日志
	InfoLevel = "info"
	// WarnLevel 表示警告级别日志
	WarnLevel = "warn"
	// ErrorLevel 表示错误级别日志
	ErrorLevel = "error"
	// PanicLevel 表示会触发panic的日志级别
	PanicLevel = "panic"
	// FatalLevel 表示会导致程序退出的日志级别
	FatalLevel = "fatal"
)

// 定义日志输出格式
const (
	// FormatJSON 表示JSON格式输出，适合生产环境
	FormatJSON = "json"
	// FormatConsole 表示控制台友好格式，适合开发环境
	FormatConsole = "console"
)

// Config 定义日志配置选项
type Config struct {
	// Level 日志级别 (debug, info, warn, error, panic, fatal)
	Level string `json:"level"`
	// Format 日志格式: json, console
	Format string `json:"format"`
	// Filename 日志文件路径
	Filename string `json:"filename"`
	// ConsoleOutput 是否同时输出到控制台
	ConsoleOutput bool `json:"console_output"`
	// EnableCaller 是否记录调用者信息
	EnableCaller bool `json:"enable_caller"`
	// EnableColor 是否启用颜色（控制台格式时有效）
	EnableColor bool `json:"enable_color"`
	// FileRotation 文件轮转配置
	FileRotation *FileRotationConfig `json:"file_rotation"`
}

// FileRotationConfig 定义日志文件轮转设置
type FileRotationConfig struct {
	// MaxSize 单个日志文件最大尺寸，单位MB
	MaxSize int `json:"max_size"`
	// MaxBackups 最多保留文件个数
	MaxBackups int `json:"max_backups"`
	// MaxAge 日志保留天数
	MaxAge int `json:"max_age"`
	// Compress 是否压缩轮转文件
	Compress bool `json:"compress"`
}

// DefaultConfig 返回默认的日志配置
func DefaultConfig() Config {
	return Config{
		Level:         InfoLevel,
		Format:        FormatConsole,
		Filename:      "./logs/app.log",
		ConsoleOutput: true,
		EnableCaller:  true,
		EnableColor:   true,
		FileRotation: &FileRotationConfig{
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 10,
			Compress:   false,
		},
	}
}

// Logger 封装 zap 日志功能的结构体
type Logger struct {
	zap         *zap.Logger
	sugar       *zap.SugaredLogger
	config      Config
	atomicLevel zap.AtomicLevel
	rotator     *lumberjack.Logger
}

// 全局默认日志实例
var defaultLogger *Logger

// parseLevel 将字符串级别转换为 zapcore.Level
func parseLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case PanicLevel:
		return zapcore.PanicLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Init 初始化默认日志器实例
// 使用提供的配置来创建全局默认日志器
func Init(config Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// NewLogger 创建新的日志器实例
// 根据提供的配置创建一个新的Logger实例
func NewLogger(config Config) (*Logger, error) {
	// 使用默认配置填充未设置的值
	defaultCfg := DefaultConfig()
	if config.Level == "" {
		config.Level = defaultCfg.Level
	}
	if config.Format == "" {
		config.Format = defaultCfg.Format
	}
	if config.Filename == "" {
		config.Filename = defaultCfg.Filename
	}
	if config.FileRotation == nil {
		config.FileRotation = defaultCfg.FileRotation
	} else {
		if config.FileRotation.MaxSize <= 0 {
			config.FileRotation.MaxSize = defaultCfg.FileRotation.MaxSize
		}
		if config.FileRotation.MaxAge <= 0 {
			config.FileRotation.MaxAge = defaultCfg.FileRotation.MaxAge
		}
		if config.FileRotation.MaxBackups <= 0 {
			config.FileRotation.MaxBackups = defaultCfg.FileRotation.MaxBackups
		}
	}

	// 创建原子级别用于动态级别变更
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(parseLevel(config.Level))

	// 设置编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 配置人类友好输出
	if config.Format == FormatConsole {
		if config.EnableColor {
			// 控制台使用彩色输出
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			// 不带颜色的大写日志级别
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		}
	}

	// 自定义时间格式
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}

	// 设置日志输出
	var writer zapcore.WriteSyncer

	// 确保日志目录存在
	dir := filepath.Dir(config.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 设置 lumberjack 进行日志轮转
	rotator := &lumberjack.Logger{
		Filename:   config.Filename,
		MaxSize:    config.FileRotation.MaxSize,
		MaxBackups: config.FileRotation.MaxBackups,
		MaxAge:     config.FileRotation.MaxAge,
		Compress:   config.FileRotation.Compress,
	}
	writer = zapcore.AddSync(rotator)

	// 添加控制台输出
	if config.ConsoleOutput {
		writer = zapcore.NewMultiWriteSyncer(writer, zapcore.AddSync(os.Stdout))
	}

	// 根据配置选择编码器
	var encoder zapcore.Encoder
	if config.Format == FormatJSON {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	// 创建核心
	core := zapcore.NewCore(encoder, writer, atomicLevel)

	// 创建 zap 日志器
	var zapLogger *zap.Logger
	if config.EnableCaller {
		zapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
	} else {
		zapLogger = zap.New(core)
	}

	logger := &Logger{
		zap:         zapLogger,
		sugar:       zapLogger.Sugar(),
		config:      config,
		atomicLevel: atomicLevel,
		rotator:     rotator,
	}

	return logger, nil
}

// SetLevel 动态更改日志级别
// 允许在运行时改变Logger的日志记录级别
func (l *Logger) SetLevel(level string) {
	l.atomicLevel.SetLevel(parseLevel(level))
}

// With 添加结构化上下文到日志器
// 返回一个带有指定字段的新Logger实例
func (l *Logger) With(fields ...zapcore.Field) *Logger {
	newZap := l.zap.With(fields...)
	return &Logger{
		zap:         newZap,
		sugar:       newZap.Sugar(),
		config:      l.config,
		atomicLevel: l.atomicLevel,
		rotator:     l.rotator,
	}
}

// WithFields 使用键值对添加结构化上下文到日志器
// 返回一个带有指定字段映射的新Logger实例
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	var zapFields []zap.Field
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return l.With(zapFields...)
}

// Debug 在 debug 级别记录消息
// 可附加结构化上下文字段
func (l *Logger) Debug(msg string, fields ...zapcore.Field) {
	l.zap.Debug(msg, fields...)
}

// Info 在 info 级别记录消息
// 可附加结构化上下文字段
func (l *Logger) Info(msg string, fields ...zapcore.Field) {
	l.zap.Info(msg, fields...)
}

// Warn 在 warn 级别记录消息
// 可附加结构化上下文字段
func (l *Logger) Warn(msg string, fields ...zapcore.Field) {
	l.zap.Warn(msg, fields...)
}

// Error 在 error 级别记录消息
// 可附加结构化上下文字段
func (l *Logger) Error(msg string, fields ...zapcore.Field) {
	l.zap.Error(msg, fields...)
}

// Panic 在 panic 级别记录消息然后触发 panic
// 可附加结构化上下文字段
func (l *Logger) Panic(msg string, fields ...zapcore.Field) {
	l.zap.Panic(msg, fields...)
}

// Fatal 在 fatal 级别记录消息然后调用 os.Exit(1)
// 可附加结构化上下文字段
func (l *Logger) Fatal(msg string, fields ...zapcore.Field) {
	l.zap.Fatal(msg, fields...)
}

// Debugf 记录格式化的 debug 级别消息
// 支持printf风格的格式化
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

// Infof 记录格式化的 info 级别消息
// 支持printf风格的格式化
func (l *Logger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

// Warnf 记录格式化的 warn 级别消息
// 支持printf风格的格式化
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

// Errorf 记录格式化的 error 级别消息
// 支持printf风格的格式化
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

// Panicf 记录格式化的 panic 级别消息然后触发 panic
// 支持printf风格的格式化
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.sugar.Panicf(format, args...)
}

// Fatalf 记录格式化的 fatal 级别消息然后调用 os.Exit(1)
// 支持printf风格的格式化
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

// Sync 刷新任何缓冲的日志条目
// 在程序退出前应调用此方法确保所有日志都已写入
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Close 正确关闭日志器
// 这是Sync方法的别名，用于遵循io.Closer接口
func (l *Logger) Close() error {
	return l.Sync()
}

// GetZapLogger 获取底层的 zap.Logger
// 当需要直接使用zap时可调用此方法
func (l *Logger) GetZapLogger() *zap.Logger {
	return l.zap
}

// GetSugarLogger 获取底层的 zap.SugaredLogger
// 当需要使用zap的Sugar API时可调用此方法
func (l *Logger) GetSugarLogger() *zap.SugaredLogger {
	return l.sugar
}

// 全局便捷函数，使用默认日志器

// SetDefaultLevel 设置默认日志器的级别
func SetDefaultLevel(level string) {
	if defaultLogger != nil {
		defaultLogger.SetLevel(level)
	}
}

// Debug 使用默认日志器记录 debug 级别消息
func Debug(msg string, fields ...zapcore.Field) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, fields...)
	}
}

// Info 使用默认日志器记录 info 级别消息
func Info(msg string, fields ...zapcore.Field) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, fields...)
	}
}

// Warn 使用默认日志器记录 warn 级别消息
func Warn(msg string, fields ...zapcore.Field) {
	if defaultLogger != nil {
		defaultLogger.Warn(msg, fields...)
	}
}

// Error 使用默认日志器记录 error 级别消息
func Error(msg string, fields ...zapcore.Field) {
	if defaultLogger != nil {
		defaultLogger.Error(msg, fields...)
	}
}

// Panic 使用默认日志器记录 panic 级别消息然后触发 panic
func Panic(msg string, fields ...zapcore.Field) {
	if defaultLogger != nil {
		defaultLogger.Panic(msg, fields...)
	}
}

// Fatal 使用默认日志器记录 fatal 级别消息然后退出
func Fatal(msg string, fields ...zapcore.Field) {
	if defaultLogger != nil {
		defaultLogger.Fatal(msg, fields...)
	}
}

// Debugf 使用默认日志器记录格式化的 debug 级别消息
func Debugf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debugf(format, args...)
	}
}

// Infof 使用默认日志器记录格式化的 info 级别消息
func Infof(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Infof(format, args...)
	}
}

// Warnf 使用默认日志器记录格式化的 warn 级别消息
func Warnf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warnf(format, args...)
	}
}

// Errorf 使用默认日志器记录格式化的 error 级别消息
func Errorf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Errorf(format, args...)
	}
}

// Panicf 使用默认日志器记录格式化的 panic 级别消息然后触发 panic
func Panicf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Panicf(format, args...)
	}
}

// Fatalf 使用默认日志器记录格式化的 fatal 级别消息然后退出
func Fatalf(format string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatalf(format, args...)
	}
}

// With 添加结构化上下文到默认日志器
func With(fields ...zapcore.Field) *Logger {
	if defaultLogger != nil {
		return defaultLogger.With(fields...)
	}
	return nil
}

// WithFields 使用键值对添加结构化上下文到默认日志器
func WithFields(fields map[string]interface{}) *Logger {
	if defaultLogger != nil {
		return defaultLogger.WithFields(fields)
	}
	return nil
}

// Sync 刷新默认日志器中任何缓冲的日志条目
func Sync() error {
	if defaultLogger != nil {
		return defaultLogger.Sync()
	}
	return nil
}

// Field 代表一个日志字段
type Field = zap.Field

// 提供常用字段类型的创建函数
var (
	// String 创建字符串类型的日志字段
	String = zap.String
	// Int 创建整数类型的日志字段
	Int = zap.Int
	// Int64 创建64位整数类型的日志字段
	Int64 = zap.Int64
	// Float64 创建浮点数类型的日志字段
	Float64 = zap.Float64
	// Bool 创建布尔类型的日志字段
	Bool = zap.Bool
	// Any 创建任意类型的日志字段
	Any = zap.Any
	// Err 从错误创建日志字段
	Err = zap.Error
	// Time 创建时间类型的日志字段
	Time = zap.Time
	// Duration 创建时间间隔类型的日志字段
	Duration = zap.Duration
)
