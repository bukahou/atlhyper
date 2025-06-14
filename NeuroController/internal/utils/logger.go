// =======================================================================================
// 📄 logger.go
//
// ✨ Description:
//     Provides a unified structured logging system based on zap. Exposes Info, Warn,
//     and Error interfaces with support for structured JSON output. Compatible with
//     log collectors like Elastic APM, Filebeat, Loki, etc.
//
// 🛠️ Features:
//     - InitLogger(): Initializes zap logger (production/development modes supported)
//     - Info(), Warn(), Error(): Unified logging methods with zap.Field support
//     - WithTraceID(): Extracts trace.id from context (for distributed tracing)
//
// 📦 Dependency:
//     - go.uber.org/zap (structured logging library)
//
// 📍 Usage:
//     - All modules should use this logger to ensure traceability and structured output
//     - Supports integration with APM and log pipeline tools
//
// ✍️ Author: bukahou（@ZGMF-X10A）
// 📅 Created: June 2025
// =======================================================================================

package utils

import (
	"context"

	"go.uber.org/zap"
)

var logger *zap.Logger

// =======================================================================================
// ✅ InitLogger
//
// 初始化全局 zap 日志记录器。
// 默认使用 zap 的生产模式，输出为 JSON 格式。
// 如果是本地开发环境，可替换为 zap.NewDevelopment()。
//
// 若创建失败会 panic（正常情况不应发生）。
func InitLogger() {
	var err error
	logger, err = zap.NewProduction() // 可替换为 zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

// =======================================================================================
// ✅ Info
//
// 打印信息级别日志。
// 接收上下文和可选的结构化字段参数。
// 用于通用的程序运行日志。
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// =======================================================================================
// ✅ Warn
//
// 打印警告级别日志。
// 用于非关键性问题或预警情况。
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

// =======================================================================================
// ✅ Error
//
// 打印错误级别日志。
// 用于记录运行错误，可携带 trace.id 和错误对象等字段。
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

// =======================================================================================
// ✅ WithTraceID
//
// 从上下文中提取 trace.id。
// 若未找到 trace ID，则返回 "unknown"。
// 用于在分布式系统中进行日志关联。
func WithTraceID(ctx context.Context) zap.Field {
	if traceID, ok := ctx.Value("trace.id").(string); ok && traceID != "" {
		return zap.String("trace.id", traceID)
	}
	return zap.String("trace.id", "unknown")
}

// =======================================================================================
// ✅ Fatal
//
// 打印致命级别日志并立即退出程序。
// 仅用于不可恢复的错误（如初始化失败）。
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}
