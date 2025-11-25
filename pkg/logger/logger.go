// Package logger 提供统一的日志封装，基于 slog，支持结构化日志、trace_id/span_id 注入、日志切割
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 是全局日志实例
var globalLogger *slog.Logger

// Config 日志配置
type Config struct {
	// 日志级别：debug, info, warn, error
	Level string `toml:"level" default:"info"`
	// 输出格式：json 或 text
	Format string `toml:"format" default:"json"`
	// 输出目标：stdout, file, both
	Output string `toml:"output" default:"stdout"`
	// 日志文件路径（当 output 为 file 或 both 时）
	FilePath string `toml:"file_path" default:"logs/app.log"`
	// 最大文件大小（MB）
	MaxSize int `toml:"max_size" default:"100"`
	// 最大备份文件数
	MaxBackups int `toml:"max_backups" default:"10"`
	// 最大保留天数
	MaxAge int `toml:"max_age" default:"30"`
	// 是否压缩
	Compress bool `toml:"compress" default:"true"`
	// 是否输出调用者信息
	WithCaller bool `toml:"with_caller" default:"true"`
}

// Init 初始化全局日志实例
func Init(cfg Config) error {
	var handler slog.Handler
	var output io.Writer

	// 设置日志级别
	var level slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// 设置日志切割
	fileWriter := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// 设置输出目标
	switch cfg.Output {
	case "file":
		output = fileWriter
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0755); err != nil {
			return err
		}
	case "both":
		// 确保日志目录存在
		if err := os.MkdirAll(filepath.Dir(cfg.FilePath), 0755); err != nil {
			return err
		}
		output = io.MultiWriter(os.Stdout, fileWriter)
	default:
		output = os.Stdout
	}

	// 设置 Handler 选项
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.WithCaller,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 可以在这里自定义字段，例如把 time 格式化
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	}

	// 创建 Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	globalLogger = slog.New(handler)
	slog.SetDefault(globalLogger)

	return nil
}

// Get 获取全局日志实例
func Get() *slog.Logger {
	if globalLogger == nil {
		return slog.Default()
	}
	return globalLogger
}

// WithContext 从 context 中提取 trace_id 和 span_id，返回带有这些字段的 logger
// 建议直接使用 InfoContext 等方法，此方法用于兼容旧习惯或链式调用
func WithContext(ctx context.Context) *slog.Logger {
	logger := Get()

	traceID := extractTraceID(ctx)
	spanID := extractSpanID(ctx)

	attrs := []any{}
	if traceID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID))
	}
	if spanID != "" {
		attrs = append(attrs, slog.String("span_id", spanID))
	}

	if len(attrs) > 0 {
		return logger.With(attrs...)
	}

	return logger
}

// Debug 输出 debug 级别日志
func Debug(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Debug(msg, args...)
}

// Info 输出 info 级别日志
func Info(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Info(msg, args...)
}

// Warn 输出 warn 级别日志
func Warn(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Warn(msg, args...)
}

// Error 输出 error 级别日志
func Error(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
}

// Fatal 输出 fatal 级别日志并退出
func Fatal(ctx context.Context, msg string, args ...any) {
	WithContext(ctx).Error(msg, args...)
	os.Exit(1)
}

// LogDuration 记录操作耗时，返回一个函数用于在 defer 中调用
func LogDuration(ctx context.Context, msg string, args ...any) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		args = append(args, slog.Duration("duration", duration))
		Info(ctx, msg, args...)
	}
}

// extractTraceID 从 context 中提取 trace_id
func extractTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value("trace_id").(string); ok && traceID != "" {
		return traceID
	}
	// TODO: 集成 OpenTelemetry
	return ""
}

// extractSpanID 从 context 中提取 span_id
func extractSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if spanID, ok := ctx.Value("span_id").(string); ok && spanID != "" {
		return spanID
	}
	// TODO: 集成 OpenTelemetry
	return ""
}
