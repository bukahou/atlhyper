// // =======================================================================================
// // 📄 logger.go
// //
// // ✨ Description:
// //     Provides a unified structured logging system based on zap. Exposes Info, Warn,
// //     and Error interfaces with support for structured JSON output. Compatible with
// //     log collectors like Elastic APM, Filebeat, Loki, etc.
// //
// // 🛠️ Features:
// //     - InitLogger(): Initializes zap logger (production/development modes supported)
// //     - Info(), Warn(), Error(): Unified logging methods with zap.Field support
// //     - WithTraceID(): Extracts trace.id from context (for distributed tracing)
// //
// // 📦 Dependency:
// //     - go.uber.org/zap (structured logging library)
// //
// // 📍 Usage:
// //     - All modules should use this logger to ensure traceability and structured output
// //     - Supports integration with APM and log pipeline tools
// //
// // ✍️ Author: bukahou（@ZGMF-X10A）
// // 📅 Created: June 2025
// // =======================================================================================

// package utils

// import (
// 	"context"
// 	"fmt"

// 	"go.uber.org/zap"
// 	"sigs.k8s.io/controller-runtime/pkg/log"
// 	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
// )

// var logger *zap.Logger

// // =======================================================================================
// // ✅ InitLogger
// //
// // 初始化全局 zap 日志记录器。
// // 默认使用 zap 的生产模式，输出为 JSON 格式。
// // 如果是本地开发环境，可替换为 zap.NewDevelopment()。
// //
// // 若创建失败会 panic（正常情况不应发生）。
// func InitLogger() {
// 	// ctrl.SetLogger(zapr.New(zapr.UseDevMode(false))) // (true): 开发模式 / (false): 生产模式

// 	// var err error
// 	// logger, err = zap.NewProduction() // 可替换为 zap.NewDevelopment()
// 	// if err != nil {
// 	// 	panic(err)
// 	// }

// 	log.SetLogger(ctrlzap.New(ctrlzap.UseDevMode(false)))

// 	fmt.Println("✅ 日志系统加载完成")
// }

// // =======================================================================================
// // ✅ Info
// //
// // 打印信息级别日志。
// // 接收上下文和可选的结构化字段参数。
// // 用于通用的程序运行日志。
// func Info(ctx context.Context, msg string, fields ...zap.Field) {
// 	logger.Info(msg, fields...)
// }

// // =======================================================================================
// // ✅ Warn
// //
// // 打印警告级别日志。
// // 用于非关键性问题或预警情况。
// func Warn(ctx context.Context, msg string, fields ...zap.Field) {
// 	logger.Warn(msg, fields...)
// }

// // =======================================================================================
// // ✅ Error
// //
// // 打印错误级别日志。
// // 用于记录运行错误，可携带 trace.id 和错误对象等字段。
// func Error(ctx context.Context, msg string, fields ...zap.Field) {
// 	logger.Error(msg, fields...)
// }

// // =======================================================================================
// // ✅ WithTraceID
// //
// // 从上下文中提取 trace.id。
// // 若未找到 trace ID，则返回 "unknown"。
// // 用于在分布式系统中进行日志关联。
// func WithTraceID(ctx context.Context) zap.Field {
// 	if traceID, ok := ctx.Value("trace.id").(string); ok && traceID != "" {
// 		return zap.String("trace.id", traceID)
// 	}
// 	return zap.String("trace.id", "unknown")
// }

// // =======================================================================================
// // ✅ Fatal
// //
// // 打印致命级别日志并立即退出程序。
// // 仅用于不可恢复的错误（如初始化失败）。
// func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
// 	logger.Fatal(msg, fields...)
// }

// =======================================================================================
// 📄 logger.go
//
// ✨ Description:
//     Provides a unified structured logging system based on logr + zap backend.
//     Automatically injects trace.id into all log entries from context.
//
// 🛠️ Features:
//     - InitLogger(): Initializes zap-based logr
//     - Info(), Warn(), Error(), Fatal(): Unified logging methods with trace.id
//     - WithTraceID(): Extracts trace.id as key-value pair for log correlation
//
// 📦 Dependency:
//     - sigs.k8s.io/controller-runtime/pkg/log
//     - go.uber.org/zap (as backend via controller-runtime)
//
// 📍 Usage:
//     - All modules use this logger via utils.* for consistent structured logging
//
// ✍️ Author: bukahou（@ZGMF-X10A）
// 📅 Created: June 2025
// =======================================================================================

// package utils

// import (
// 	"context"
// 	"fmt"

// 	"sigs.k8s.io/controller-runtime/pkg/log"
// 	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
// )

// // =======================================================================================
// // ✅ InitLogger
// //
// // 初始化全局日志系统（zap-based logr）。用于 controller-runtime。
// func InitLogger() {
// 	log.SetLogger(ctrlzap.New(ctrlzap.UseDevMode(true))) // 设置为 DevMode(true) 可读性更高，适合本地调试（false生产模式）
// 	fmt.Println("✅ 日志系统加载完成")
// }

// // =======================================================================================
// // ✅ Info
// //
// // Info 日志封装，自动带上 trace.id（如有）。
// func Info(ctx context.Context, msg string, keysAndValues ...interface{}) {
// 	log.FromContext(ctx).WithValues(WithTraceID(ctx)...).Info(msg, keysAndValues...)
// }

// // =======================================================================================
// // ✅ Warn
// //
// // Warn 日志封装，通过 V(1) 表示较低优先级日志（logr 标准方式）。
// func Warn(ctx context.Context, msg string, keysAndValues ...interface{}) {
// 	log.FromContext(ctx).WithValues(WithTraceID(ctx)...).V(1).Info("[WARN] "+msg, keysAndValues...)
// }

// // =======================================================================================
// // ✅ Error
// //
// // Error 日志封装，支持错误对象与 trace.id。
// func Error(ctx context.Context, msg string, err error, keysAndValues ...interface{}) {
// 	log.FromContext(ctx).WithValues(WithTraceID(ctx)...).Error(err, msg, keysAndValues...)
// }

// // =======================================================================================
// // ✅ Fatal
// //
// // Fatal 日志封装，记录错误后终止程序。
// // logr 不支持 Fatal 级别，因此手动 panic。
// func Fatal(ctx context.Context, msg string, err error, keysAndValues ...interface{}) {
// 	log.FromContext(ctx).WithValues(WithTraceID(ctx)...).Error(err, "[FATAL] "+msg, keysAndValues...)
// 	panic(err)
// }

// // =======================================================================================
// // ✅ WithTraceID
// //
// // 提取 trace.id，用于日志链路追踪（上下游一致性）。
// func WithTraceID(ctx context.Context) []interface{} {
// 	if traceID, ok := ctx.Value("trace.id").(string); ok && traceID != "" {
// 		return []interface{}{"trace.id", traceID}
// 	}
// 	return []interface{}{"trace.id", "unknown"}
// }

//	func TraceArgs(ctx context.Context, kvs ...interface{}) []interface{} {
//		return append(WithTraceID(ctx), kvs...)
//	}
package utils

// ctrl.SetLogger(zap.New(zap.UseDevMode(true)))
