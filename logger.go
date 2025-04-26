// Package logger 提供一个灵活的日志系统，基于 uber-go/zap
// 支持结构化日志和人类友好的输出格式
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 定义日志级别常量
const (
	DebugLevel = "debug"
	InfoLevel  = "info"
	WarnLevel  = "warn"
	ErrorLevel = "error"
	PanicLevel = "panic"
	FatalLevel = "fatal"
)

// 定义日志输出格式
const (
	FormatJSON    = "json"    // JSON格式，适合生产环境
	FormatConsole = "console" // 控制台友好格式，适合开发环境
)

// Config 定义日志配置
type Config struct {
	// 日志级别 (debug, info, warn, error, panic, fatal)
	Level string `json:"level"`
	// 日志格式: json, console
	Format string `json:"format"`
	// 日志文件路径
	Filename string `json:"filename"`
	// 是否输出到控制台
	ConsoleOutput bool `json:"console_output"`
	// 是否记录调用者信息
	EnableCaller bool `json:"enable_caller"`
	// 是否启用颜色（控制台格式时有效）
	EnableColor bool `json:"enable_color"`
	// 文件轮转配置
	FileRotation *FileRotationConfig `json:"file_rotation"`
}

// FileRotationConfig 定义日志文件轮转设置
type FileRotationConfig struct {
	// 单个日志文件最大尺寸，单位MB
	MaxSize int `json:"max_size"`
	// 最多保留文件个数
	MaxBackups int `json:"max_backups"`
	// 日志保留天数
	MaxAge int `json:"max_age"`
	// 是否压缩轮转文件
	Compress bool `json:"compress"`
}

// DefaultConfig 返回默认配置
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

// Logger 封装 zap 日志功能
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

// Init 初始化默认日志器
func Init(config Config) error {
	logger, err := NewLogger(config)
	if err != nil {
		return err
	}
	defaultLogger = logger
	return nil
}

// NewLogger 创建新的日志器实例
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
	if config.Format == FormatConsole && config.EnableColor {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
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
		zapLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
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

// 添加源代码上下文（文件，行）
func addSourceContext(fields []zapcore.Field) []zapcore.Field {
	_, file, line, ok := runtime.Caller(2)
	if ok {
		// 从完整路径中获取文件名
		_, filename := filepath.Split(file)
		sourceInfo := fmt.Sprintf("%s:%d", filename, line)

		// 添加源代码上下文作为字段
		fields = append(fields, zap.String("source", sourceInfo))
	}
	return fields
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
	String   = zap.String
	Int      = zap.Int
	Int64    = zap.Int64
	Float64  = zap.Float64
	Bool     = zap.Bool
	Any      = zap.Any
	Err      = zap.Error
	Time     = zap.Time
	Duration = zap.Duration
)

// TracedLogger 带跟踪ID的日志接口
type TracedLogger struct {
	traceID string
	logger  *Logger
}

// NewTracedLogger 创建带跟踪ID的日志器
func NewTracedLogger(traceID string) *TracedLogger {
	if defaultLogger == nil {
		return nil
	}
	return &TracedLogger{
		traceID: traceID,
		logger:  defaultLogger.With(zap.String("trace_id", traceID)),
	}
}

// Debug 输出带跟踪ID的Debug级别日志
func (t *TracedLogger) Debug(msg string, fields ...Field) {
	t.logger.Debug(msg, fields...)
}

// Info 输出带跟踪ID的Info级别日志
func (t *TracedLogger) Info(msg string, fields ...Field) {
	t.logger.Info(msg, fields...)
}

// Warn 输出带跟踪ID的Warn级别日志
func (t *TracedLogger) Warn(msg string, fields ...Field) {
	t.logger.Warn(msg, fields...)
}

// Error 输出带跟踪ID的Error级别日志
func (t *TracedLogger) Error(msg string, fields ...Field) {
	t.logger.Error(msg, fields...)
}

// Panic 输出带跟踪ID的Panic级别日志
func (t *TracedLogger) Panic(msg string, fields ...Field) {
	t.logger.Panic(msg, fields...)
}

// Fatal 输出带跟踪ID的Fatal级别日志
func (t *TracedLogger) Fatal(msg string, fields ...Field) {
	t.logger.Fatal(msg, fields...)
}

// Debugf 输出带跟踪ID的格式化Debug级别日志
func (t *TracedLogger) Debugf(format string, args ...interface{}) {
	t.logger.Debugf(format, args...)
}

// Infof 输出带跟踪ID的格式化Info级别日志
func (t *TracedLogger) Infof(format string, args ...interface{}) {
	t.logger.Infof(format, args...)
}

// Warnf 输出带跟踪ID的格式化Warn级别日志
func (t *TracedLogger) Warnf(format string, args ...interface{}) {
	t.logger.Warnf(format, args...)
}

// Errorf 输出带跟踪ID的格式化Error级别日志
func (t *TracedLogger) Errorf(format string, args ...interface{}) {
	t.logger.Errorf(format, args...)
}

// Panicf 输出带跟踪ID的格式化Panic级别日志
func (t *TracedLogger) Panicf(format string, args ...interface{}) {
	t.logger.Panicf(format, args...)
}

// Fatalf 输出带跟踪ID的格式化Fatal级别日志
func (t *TracedLogger) Fatalf(format string, args ...interface{}) {
	t.logger.Fatalf(format, args...)
}
