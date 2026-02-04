// Package logger 提供统一的结构化日志功能
// 基于 Go 1.21+ log/slog 标准库
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	defaultLogger *slog.Logger
	once          sync.Once
)

// Config 日志配置
type Config struct {
	Level  string // debug/info/warn/error，默认 info
	Format string // text/json，默认 text
	Output io.Writer // 输出目标，默认 os.Stdout
}

// Init 初始化全局日志器
// 应在 main() 开始时调用一次
func Init(cfg Config) {
	once.Do(func() {
		initLogger(cfg)
	})
}

func initLogger(cfg Config) {
	// 解析日志级别
	level := parseLevel(cfg.Level)

	// 输出目标
	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	// 创建 Handler
	opts := &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 简化时间格式
			if a.Key == slog.TimeKey {
				t := a.Value.Time()
				a.Value = slog.StringValue(t.Format("15:04:05"))
			}
			return a
		},
	}

	var handler slog.Handler
	if strings.ToLower(cfg.Format) == "json" {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// 确保日志器已初始化
func ensureInit() {
	if defaultLogger == nil {
		Init(Config{})
	}
}

// Module 创建带模块标签的日志器
func Module(name string) *ModuleLogger {
	ensureInit()
	return &ModuleLogger{
		name:   name,
		logger: defaultLogger.With("module", name),
	}
}

// ModuleLogger 模块日志器
type ModuleLogger struct {
	name   string
	logger *slog.Logger
}

// Debug 调试日志（周期性任务成功、详细追踪）
func (m *ModuleLogger) Debug(msg string, args ...any) {
	m.logger.Debug(msg, args...)
}

// Info 信息日志（关键业务事件、状态变化）
func (m *ModuleLogger) Info(msg string, args ...any) {
	m.logger.Info(msg, args...)
}

// Warn 警告日志（可恢复的异常）
func (m *ModuleLogger) Warn(msg string, args ...any) {
	m.logger.Warn(msg, args...)
}

// Error 错误日志（需要关注的错误）
func (m *ModuleLogger) Error(msg string, args ...any) {
	m.logger.Error(msg, args...)
}

// With 添加上下文字段
func (m *ModuleLogger) With(args ...any) *ModuleLogger {
	return &ModuleLogger{
		name:   m.name,
		logger: m.logger.With(args...),
	}
}

// WithContext 从 context 中提取追踪信息
func (m *ModuleLogger) WithContext(ctx context.Context) *ModuleLogger {
	newLogger := m.logger

	// 提取 request_id
	if reqID := ctx.Value(CtxKeyRequestID); reqID != nil {
		newLogger = newLogger.With("request_id", reqID)
	}

	// 提取 user_id
	if userID := ctx.Value(CtxKeyUserID); userID != nil {
		newLogger = newLogger.With("user_id", userID)
	}

	return &ModuleLogger{
		name:   m.name,
		logger: newLogger,
	}
}

// 上下文 Key 定义
type ctxKey string

const (
	CtxKeyRequestID ctxKey = "request_id"
	CtxKeyUserID    ctxKey = "user_id"
)

// ContextWithRequestID 在 context 中设置 request_id
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, CtxKeyRequestID, requestID)
}

// ContextWithUserID 在 context 中设置 user_id
func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, CtxKeyUserID, userID)
}

// ----- 便捷全局函数 -----

// Debug 全局调试日志
func Debug(msg string, args ...any) {
	ensureInit()
	defaultLogger.Debug(msg, args...)
}

// Info 全局信息日志
func Info(msg string, args ...any) {
	ensureInit()
	defaultLogger.Info(msg, args...)
}

// Warn 全局警告日志
func Warn(msg string, args ...any) {
	ensureInit()
	defaultLogger.Warn(msg, args...)
}

// Error 全局错误日志
func Error(msg string, args ...any) {
	ensureInit()
	defaultLogger.Error(msg, args...)
}

// ----- 辅助函数 -----

// Duration 格式化耗时为易读形式
func Duration(d time.Duration) string {
	if d < time.Millisecond {
		return d.String()
	}
	if d < time.Second {
		return d.Round(time.Microsecond).String()
	}
	return d.Round(time.Millisecond).String()
}
