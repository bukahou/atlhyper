// 通用格式化工具

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
