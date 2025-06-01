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
//     - Info(), Error(): 日志输出接口，支持可选 zap.Field 扩展
//     - WithTraceID(): 从 context 中提取 trace.id 字段（预留链路追踪扩展）
//
// 📦 依赖：
//     - go.uber.org/zap
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

// InitLogger 初始化日志系统
func InitLogger() {
	var err error
	logger, err = zap.NewProduction() // 或 zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

// Info 输出 info 级别日志（支持传入 traceID 等字段）
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// Error 输出 error 级别日志
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

// WithTraceID 可选封装：从 context 中提取 traceID 字段
func WithTraceID(ctx context.Context) zap.Field {
	// 示例：你可以从 ctx 中解析 trace.id（若你使用了 apm.ContextWithTransaction 等）
	return zap.String("trace.id", ctx.Value("trace.id").(string))
}

// Warn 输出 warn 级别日志
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}
