// 通用格式化工具

/**
 * 毫秒持续时间格式化（APM 用）
 * 0.5ms -> "0.5ms", 1234ms -> "1.23s", 0.1ms -> "100μs"
 */
export function formatDurationMs(ms: number): string {
  if (ms < 0.001) return "0ms";
  if (ms < 1) return `${Math.round(ms * 1000)}μs`;
  if (ms < 1000) return `${ms.toFixed(ms < 10 ? 1 : 0)}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  return `${(ms / 60000).toFixed(1)}min`;
}

/**
 * ISO 时间戳相对时间
 */
export function formatTimeAgo(isoTimestamp: string): string {
  const diffMs = Date.now() - new Date(isoTimestamp).getTime();
  const diffMin = diffMs / 60000;
  const diffHour = diffMin / 60;
  const diffDay = diffHour / 24;

  if (diffDay >= 1) return `${Math.floor(diffDay)}d ago`;
  if (diffHour >= 1) return `${Math.floor(diffHour)}h ago`;
  if (diffMin >= 1) return `${Math.floor(diffMin)}m ago`;
  return "just now";
}

export function formatBytes(bytes: number, decimals = 2): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ["B", "KB", "MB", "GB", "TB", "PB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + " " + sizes[i];
}

export function formatBytesPS(bytesPS: number): string {
  return formatBytes(bytesPS) + "/s";
}

export function formatNumber(num: number, decimals = 1): string {
  if (num >= 1000000) {
    return (num / 1000000).toFixed(decimals) + "M";
  }
  if (num >= 1000) {
    return (num / 1000).toFixed(decimals) + "K";
  }
  return num.toFixed(decimals);
}
