// Package clog 提供一个灵活的日志系统，基于 uber-go/zap
// 支持结构化日志、多环境配置、多日志器管理和日志文件轮转
package clog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 日志级别常量定义
const (
	DebugLevel = "debug" // 调试级别日志
	InfoLevel  = "info"  // 信息级别日志
	WarnLevel  = "warn"  // 警告级别日志
	ErrorLevel = "error" // 错误级别日志
	PanicLevel = "panic" // 会触发panic的日志级别
	FatalLevel = "fatal" // 会导致程序退出的日志级别
)

// 日志输出格式定义
const (
	FormatJSON    = "json"    // JSON格式输出，适合生产环境
	FormatConsole = "console" // 控制台友好格式，适合开发环境
)

// 运行环境类型定义
const (
	EnvDevelopment = "development" // 开发环境
	EnvProduction  = "production"  // 生产环境
	EnvTest        = "test"        // 测试环境
)

// 默认运行环境
var currentEnv = EnvDevelopment

// 全局默认日志实例
var defaultLogger *Logger

// 多日志器映射表
var (
	loggers   = make(map[string]*Logger)
	loggersMu sync.RWMutex
)

// Config 定义日志配置选项
type Config struct {
	Level                string              `json:"level"`                  // 日志级别
	Format               string              `json:"format"`                 // 日志格式
	Filename             string              `json:"filename"`               // 日志文件路径
	Name                 string              `json:"name"`                   // 日志器名称
	ConsoleOutput        bool                `json:"console_output"`         // 是否同时输出到控制台
	EnableCaller         bool                `json:"enable_caller"`          // 是否记录调用者信息
	EnableColor          bool                `json:"enable_color"`           // 是否启用颜色
	FileRotation         *FileRotationConfig `json:"file_rotation"`          // 文件轮转配置
	Environment          string              `json:"environment"`            // 运行环境
	UseTimeStampFilename bool                `json:"use_timestamp_filename"` // 是否使用时间戳文件名
	UsePidFilename       bool                `json:"use_pid_filename"`       // 是否在文件名中包含进程ID
}

// FileRotationConfig 定义日志文件轮转设置
type FileRotationConfig struct {
	MaxSize    int  `json:"max_size"`    // 单个日志文件最大尺寸(MB)
	MaxBackups int  `json:"max_backups"` // 最多保留文件个数
	MaxAge     int  `json:"max_age"`     // 日志保留天数
	Compress   bool `json:"compress"`    // 是否压缩轮转文件
}

// Logger 封装 zap 日志功能的结构体
type Logger struct {
	zap         *zap.Logger        // 底层zap日志器
	sugar       *zap.SugaredLogger // 语法糖日志器
	config      Config             // 日志配置
	atomicLevel zap.AtomicLevel    // 原子级别控制
	rotator     *lumberjack.Logger // 日志轮转器
}

// Field 代表一个日志字段
type Field = zap.Field

// 提供常用字段类型的创建函数
var (
	String   = zap.String   // 创建字符串类型的日志字段
	Int      = zap.Int      // 创建整数类型的日志字段
	Int64    = zap.Int64    // 创建64位整数类型的日志字段
	Float64  = zap.Float64  // 创建浮点数类型的日志字段
	Bool     = zap.Bool     // 创建布尔类型的日志字段
	Any      = zap.Any      // 创建任意类型的日志字段
	Err      = zap.Error    // 从错误创建日志字段
	Time     = zap.Time     // 创建时间类型的日志字段
	Duration = zap.Duration // 创建时间间隔类型的日志字段
)

// DefaultConfig 返回默认的日志配置
func DefaultConfig() Config {
	return Config{
		Level:                InfoLevel,
		Format:               FormatConsole,
		Filename:             "./logs/app.log",
		Name:                 "default",
		ConsoleOutput:        false,
		EnableCaller:         true,
		EnableColor:          true,
		Environment:          currentEnv,
		UseTimeStampFilename: false,
		UsePidFilename:       false,
		FileRotation: &FileRotationConfig{
			MaxSize:    100,
			MaxAge:     7,
			MaxBackups: 10,
			Compress:   false,
		},
	}
}

// SetEnvironment 设置全局运行环境
func SetEnvironment(env string) {
	currentEnv = env
}

// GetEnvironment 获取当前运行环境
func GetEnvironment() string {
	return currentEnv
}

// Init 初始化默认日志器实例
// 使用提供的配置来创建全局默认日志器
func Init(config Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	defaultLogger = logger

	// 添加到日志器映射表
	loggersMu.Lock()
	loggers["default"] = logger
	loggersMu.Unlock()

	return nil
}

// NewLogger 创建新的日志器实例
// 根据提供的配置创建一个新的Logger实例
func NewLogger(config Config) (*Logger, error) {
	// 填充未设置的配置值
	config = fillDefaultConfig(config)

	// 创建原子级别控制
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(parseLevel(config.Level))

	// 创建编码器配置
	encoderConfig := createEncoderConfig(config)

	// 准备日志写入器
	finalFilename := getLogFilename(config)
	fileWriter, rotator, err := createLogWriter(finalFilename, config)
	if err != nil {
		return nil, err
	}

	// 为文件创建无颜色的编码器配置
	fileEncoderConfig := encoderConfig
	if config.Format == FormatConsole {
		fileEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 文件输出不使用颜色
	}

	// 为文件选择编码器
	fileEncoder := createEncoder(config, fileEncoderConfig)

	// 创建核心组件
	var core zapcore.Core
	core = zapcore.NewCore(fileEncoder, fileWriter, atomicLevel)

	// 如果需要控制台输出，创建独立的控制台编码器
	if config.ConsoleOutput {
		// 控制台编码器可以使用颜色
		consoleEncoderConfig := encoderConfig
		consoleEncoder := createEncoder(config, consoleEncoderConfig)
		consoleWriter := zapcore.AddSync(os.Stdout)

		// 合并文件核心和控制台核心
		core = zapcore.NewTee(
			core,
			zapcore.NewCore(consoleEncoder, consoleWriter, atomicLevel),
		)
	}

	// 创建和配置zap日志器
	zapLogger := createZapLogger(core, config)

	// 创建Logger实例
	logger := &Logger{
		zap:         zapLogger,
		sugar:       zapLogger.Sugar(),
		config:      config,
		atomicLevel: atomicLevel,
		rotator:     rotator,
	}

	// 添加到日志器映射表
	registerLogger(logger, config.Name)

	return logger, nil
}

// fillDefaultConfig 用默认值填充未设置的配置项
func fillDefaultConfig(config Config) Config {
	defaultCfg := DefaultConfig()

	// 填充基本配置
	if config.Level == "" {
		config.Level = defaultCfg.Level
	}
	if config.Format == "" {
		config.Format = defaultCfg.Format
	}
	if config.Filename == "" {
		config.Filename = defaultCfg.Filename
	}
	if config.Name == "" {
		config.Name = defaultCfg.Name
	}
	if config.Environment == "" {
		config.Environment = defaultCfg.Environment
	}

	// 填充文件轮转配置
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

	return config
}

// createEncoderConfig 创建编码器配置
func createEncoderConfig(config Config) zapcore.EncoderConfig {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
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
			encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		}
	}

	// 自定义时间格式
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}

	return encoderConfig
}

// createLogWriter 创建日志文件写入器
// 注意: 如果配置了 ConsoleOutput 为 true，控制台输出会在 NewLogger 函数中单独处理
func createLogWriter(filename string, config Config) (zapcore.WriteSyncer, *lumberjack.Logger, error) {
	// 确保日志目录存在
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 设置日志轮转
	rotator := &lumberjack.Logger{
		Filename:   filename,
		MaxSize:    config.FileRotation.MaxSize,
		MaxBackups: config.FileRotation.MaxBackups,
		MaxAge:     config.FileRotation.MaxAge,
		Compress:   config.FileRotation.Compress,
	}
	writer := zapcore.AddSync(rotator)

	// 不再在这里添加控制台输出，移到了 NewLogger 函数中
	return writer, rotator, nil
}

// createEncoder 根据配置创建编码器
func createEncoder(config Config, encoderConfig zapcore.EncoderConfig) zapcore.Encoder {
	if config.Format == FormatJSON {
		return zapcore.NewJSONEncoder(encoderConfig)
	}
	return zapcore.NewConsoleEncoder(encoderConfig)
}

// createZapLogger 创建并配置zap日志器
func createZapLogger(core zapcore.Core, config Config) *zap.Logger {
	var zapLogger *zap.Logger
	if config.EnableCaller {
		zapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2))
	} else {
		zapLogger = zap.New(core)
	}

	// 添加日志器名称
	if config.Name != "" {
		zapLogger = zapLogger.Named(config.Name)
	}

	return zapLogger
}

// registerLogger 注册日志器到全局映射表
func registerLogger(logger *Logger, name string) {
	if name != "" && name != "default" {
		loggersMu.Lock()
		loggers[name] = logger
		loggersMu.Unlock()
	}
}

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

// getLogFilename 根据配置生成日志文件名
func getLogFilename(config Config) string {
	// 如果不是开发环境或没有启用特殊文件名，直接返回配置的文件名
	if config.Environment != EnvDevelopment || (!config.UseTimeStampFilename && !config.UsePidFilename) {
		return config.Filename
	}

	// 解析文件路径
	dir := filepath.Dir(config.Filename)
	ext := filepath.Ext(config.Filename)
	base := filepath.Base(config.Filename)
	name := strings.TrimSuffix(base, ext)

	// 生成文件名
	var filename string
	if config.UseTimeStampFilename {
		timestamp := time.Now().Format("20060102_150405")
		filename = fmt.Sprintf("%s_%s", name, timestamp)
	} else {
		filename = name
	}

	// 加入进程ID
	if config.UsePidFilename {
		pid := os.Getpid()
		filename = fmt.Sprintf("%s_pid%d", filename, pid)
	}

	return filepath.Join(dir, filename+ext)
}

// GetLogger 获取命名日志器实例
// 如果指定名称的日志器不存在，则返回默认日志器
func GetLogger(name string) *Logger {
	loggersMu.RLock()
	defer loggersMu.RUnlock()

	logger, ok := loggers[name]
	if !ok {
		return defaultLogger
	}
	return logger
}

// Module 创建或获取模块专用日志器
// 为不同模块创建专用日志器的便捷方法
func Module(moduleName string, config ...Config) *Logger {
	// 检查是否已存在同名日志器
	loggersMu.RLock()
	logger, exists := loggers[moduleName]
	loggersMu.RUnlock()

	if exists {
		return logger
	}

	// 准备配置
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	} else {
		cfg = DefaultConfig()
	}

	// 设置模块名
	cfg.Name = moduleName

	// 使用模块名作为日志文件名
	if cfg.Filename == "" || cfg.Filename == DefaultConfig().Filename {
		dir := "./logs"
		cfg.Filename = filepath.Join(dir, moduleName+".log")
	}

	// 创建日志器
	logger, err := NewLogger(cfg)
	if err != nil {
		// 创建失败时使用默认日志器
		if defaultLogger != nil {
			defaultLogger.Error("创建模块日志器失败",
				String("module", moduleName),
				Err(err))
		}
		return defaultLogger
	}

	return logger
}

//
// Logger 实例方法
//

// SetLevel 动态更改日志级别
func (l *Logger) SetLevel(level string) {
	l.atomicLevel.SetLevel(parseLevel(level))
}

// With 添加结构化上下文到日志器
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
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	var zapFields []zap.Field
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return l.With(zapFields...)
}

// Debug 在 debug 级别记录消息
func (l *Logger) Debug(msg string, fields ...zapcore.Field) {
	l.zap.Debug(msg, fields...)
}

// Info 在 info 级别记录消息
func (l *Logger) Info(msg string, fields ...zapcore.Field) {
	l.zap.Info(msg, fields...)
}

// Warn 在 warn 级别记录消息
func (l *Logger) Warn(msg string, fields ...zapcore.Field) {
	l.zap.Warn(msg, fields...)
}

// Error 在 error 级别记录消息
func (l *Logger) Error(msg string, fields ...zapcore.Field) {
	l.zap.Error(msg, fields...)
}

// Panic 在 panic 级别记录消息然后触发 panic
func (l *Logger) Panic(msg string, fields ...zapcore.Field) {
	l.zap.Panic(msg, fields...)
}

// Fatal 在 fatal 级别记录消息然后调用 os.Exit(1)
func (l *Logger) Fatal(msg string, fields ...zapcore.Field) {
	l.zap.Fatal(msg, fields...)
}

// Debugf 记录格式化的 debug 级别消息
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

// Infof 记录格式化的 info 级别消息
func (l *Logger) Infof(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

// Warnf 记录格式化的 warn 级别消息
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

// Errorf 记录格式化的 error 级别消息
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

// Panicf 记录格式化的 panic 级别消息然后触发 panic
func (l *Logger) Panicf(format string, args ...interface{}) {
	l.sugar.Panicf(format, args...)
}

// Fatalf 记录格式化的 fatal 级别消息然后调用 os.Exit(1)
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}

// Sync 刷新任何缓冲的日志条目
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Close 正确关闭日志器
func (l *Logger) Close() error {
	return l.Sync()
}

// GetZapLogger 获取底层的 zap.Logger
func (l *Logger) GetZapLogger() *zap.Logger {
	return l.zap
}

// GetSugarLogger 获取底层的 zap.SugaredLogger
func (l *Logger) GetSugarLogger() *zap.SugaredLogger {
	return l.sugar
}

//
// 全局便捷函数
//

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

// SyncAll 刷新所有日志器中的缓冲日志条目
func SyncAll() {
	loggersMu.RLock()
	defer loggersMu.RUnlock()

	for _, logger := range loggers {
		_ = logger.Sync()
	}
}
