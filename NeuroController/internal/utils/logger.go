// =======================================================================================
// 📄 logger.go
//
// ✨ 功能说明：
//     封装统一的结构化日志系统，基于 zap 实现。提供 Info、Error 等接口，
//     支持输出 JSON 格式日志，方便与 Elastic APM、Filebeat、Loki 等日志系统集成。
//     所有模块均应通过此日志系统进行输出，便于链路追踪与模块分析。
//
// 🛠️ 提供功能：
//     - InitLogger(): 初始化 zap 日志（支持生产/开发模式）
//     - Info(), Warn(), Error(): 日志输出接口，支持可选 zap.Field 扩展
//     - WithTraceID(): 从 context 中提取 trace.id 字段（预留链路追踪扩展）
//
// 📦 依赖：
//     - go.uber.org/zap（结构化日志库）
//
// 📍 使用场景：
//     - 所有模块调用统一日志接口进行输出，支持 traceID / module 字段注入
//     - 与 APM 工具联动，进行调用链日志分析
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package utils

import (
	"context"

	"go.uber.org/zap"
)

var logger *zap.Logger

// =======================================================================================
// ✅ 方法：InitLogger
//
// 初始化日志系统，默认启用 zap 的生产模式（JSON 输出），
// 若需切换为开发模式（控制台日志），可替换为 zap.NewDevelopment()。
//
// 初始化失败将 panic 终止程序（通常不会发生）。
func InitLogger() {
	var err error
	logger, err = zap.NewProduction() // 或 zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

// =======================================================================================
// ✅ 方法：Info
//
// 输出 info 级别日志，支持注入 context 中的 traceID 等结构化字段。
// 建议所有信息级别日志统一调用该函数。
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// =======================================================================================
// ✅ 方法：Warn
//
// 输出 warn 级别日志，支持结构化字段，
// 通常用于潜在问题或告警信息（不致命错误）。
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

// =======================================================================================
// ✅ 方法：Error
//
// 输出 error 级别日志，适用于明确错误场景，
// 支持附带 traceID、error string、对象字段等结构化日志。
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

// =======================================================================================
// ✅ 方法：WithTraceID
//
// 从 context.Context 中提取 trace.id 字段（需事先注入），
// 若未找到则返回默认 "unknown" 字段，避免 panic。
// 常用于日志追踪链路统一标识。
func WithTraceID(ctx context.Context) zap.Field {
	if traceID, ok := ctx.Value("trace.id").(string); ok && traceID != "" {
		return zap.String("trace.id", traceID)
	}
	return zap.String("trace.id", "unknown")
}

// =======================================================================================
// ✅ 方法：Fatal
//
// 输出 fatal 级别日志（致命错误），记录日志后立即 os.Exit(1) 终止程序。
// 通常用于初始化失败、无法恢复的错误。
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}
