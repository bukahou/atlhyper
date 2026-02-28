/**
 * 可观测性查询 API — 桶文件（re-export）
 *
 * 按信号域拆分为独立文件，本文件提供兼容导入。
 *
 * 子文件:
 *   - observe-common.ts   — 共享类型
 *   - observe-metrics.ts  — Metrics API
 *   - observe-logs.ts     — Logs API
 *   - observe-apm.ts      — APM (Traces) API
 *   - observe-slo.ts      — SLO API
 *   - observe-health.ts   — Landing Page API
 */

export * from "./observe-common";
export * from "./observe-metrics";
export * from "./observe-logs";
export * from "./observe-apm";
export * from "./observe-slo";
export * from "./observe-health";
