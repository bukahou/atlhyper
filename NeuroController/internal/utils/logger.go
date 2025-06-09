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
// Initializes the global zap logger.
// Defaults to zap's production mode with JSON output.
// For local development, replace with zap.NewDevelopment().
//
// Panics if logger creation fails (should not normally occur).
func InitLogger() {
	var err error
	logger, err = zap.NewProduction() // or zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
}

// =======================================================================================
// ✅ Info
//
// Logs an informational message.
// Accepts context and optional structured fields.
// Use this for general application logs.
func Info(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

// =======================================================================================
// ✅ Warn
//
// Logs a warning-level message.
// Use this for non-critical issues or alerting conditions.
func Warn(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

// =======================================================================================
// ✅ Error
//
// Logs an error-level message.
// Intended for operational errors, with support for trace.id and error objects.
func Error(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

// =======================================================================================
// ✅ WithTraceID
//
// Extracts the trace.id from context.
// Returns "unknown" if trace ID is not found.
// Used for log correlation across distributed systems.
func WithTraceID(ctx context.Context) zap.Field {
	if traceID, ok := ctx.Value("trace.id").(string); ok && traceID != "" {
		return zap.String("trace.id", traceID)
	}
	return zap.String("trace.id", "unknown")
}

// =======================================================================================
// ✅ Fatal
//
// Logs a fatal-level message and exits the program.
// Use this only for unrecoverable errors (e.g., failed initialization).
func Fatal(ctx context.Context, msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}
