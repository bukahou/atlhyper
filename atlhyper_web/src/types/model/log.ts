/**
 * Log 数据模型 — 对齐 model_v3/log/log.go
 *
 * 数据源: ClickHouse otel_logs 表
 * JSON tag 统一 camelCase，前端直接使用后端字段名。
 */

// ============================================================
// LogEntry — otel_logs 行的领域模型
// ============================================================

export interface LogEntry {
  timestamp: string;              // ISO 8601
  traceId: string;                // 关联 trace（空字符串=无关联）
  spanId: string;
  severity: string;               // "INFO" | "DEBUG" | "WARN" | "ERROR"
  severityNum: number;            // 9=INFO, 5=DEBUG, 13=WARN, 17=ERROR
  serviceName: string;
  body: string;                   // 日志正文
  scopeName: string;              // 日志来源类全名
  attributes: Record<string, string>;
  resource: Record<string, string>;
}

// ============================================================
// Facets — 分面统计
// ============================================================

export interface LogFacet {
  value: string;
  count: number;
}

export interface LogFacets {
  services: LogFacet[];
  severities: LogFacet[];
  scopes: LogFacet[];
}

// ============================================================
// QueryResult — 日志搜索结果
// ============================================================

export interface LogQueryResult {
  logs: LogEntry[];
  total: number;
  facets: LogFacets;
}

// ============================================================
// Helper functions
// ============================================================

export function hasTrace(entry: LogEntry): boolean {
  return entry.traceId !== "";
}

/** 取最后一段类名: com.geass.gateway.filter.AuthVerifyFilter → AuthVerifyFilter */
export function shortScopeName(full: string): string {
  const idx = full.lastIndexOf(".");
  return idx >= 0 ? full.substring(idx + 1) : full;
}

/** 级别 → 颜色 class */
export function severityColor(severity: string): string {
  switch (severity.toUpperCase()) {
    case "ERROR": return "bg-red-500/10 text-red-500";
    case "WARN": return "bg-amber-500/10 text-amber-500";
    case "DEBUG": return "bg-gray-500/10 text-gray-500";
    case "INFO":
    default: return "bg-blue-500/10 text-blue-500";
  }
}
