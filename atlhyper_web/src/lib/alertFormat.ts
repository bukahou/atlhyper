/**
 * 告警格式化工具
 *
 * 将告警列表格式化为 AI 可读的消息文本
 */

import type { RecentAlert } from "@/types/overview";

/**
 * 将告警列表格式化为 AI 分析消息
 */
export function formatAlertsMessage(alerts: RecentAlert[]): string {
  if (alerts.length === 0) return "";

  const lines: string[] = [
    "[以下是用户选择的告警信息，请分析并给出诊断结论]",
    "",
  ];

  alerts.forEach((alert, idx) => {
    lines.push(`告警 ${idx + 1}:`);
    lines.push(`- 资源类型: ${alert.kind}`);
    lines.push(`- 命名空间: ${alert.namespace}`);
    lines.push(`- 资源名称: ${alert.name}`);
    lines.push(`- 告警原因: ${alert.reason}`);
    lines.push(`- 告警消息: ${alert.message}`);
    lines.push(`- 发生时间: ${alert.timestamp}`);
    lines.push("");
  });

  lines.push("请查询相关资源的状态、日志和事件，分析告警原因并给出修复建议。");

  return lines.join("\n");
}
