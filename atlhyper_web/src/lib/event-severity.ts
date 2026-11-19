// 事件严重程度 → 徽章颜色与文案（只有这一份）。严重程度由后端判定（含 critical），前端不再自己推断
import type { EventSeverity } from "@/types/cluster";

type BadgeType = "success" | "warning" | "error" | "info" | "default";

interface SeverityLabels {
  criticalEvents: string;
  warningEvents: string;
  normalEvents: string;
}

/** 严重程度 → StatusBadge 的颜色类型 */
export function eventSeverityBadge(severity: EventSeverity): BadgeType {
  switch (severity) {
    case "critical":
    case "error":
      return "error";
    case "warning":
      return "warning";
    default:
      return "info";
  }
}

/** 严重程度 → 显示文案（error 与 critical 同属故障级） */
export function eventSeverityLabel(severity: EventSeverity, t: SeverityLabels): string {
  switch (severity) {
    case "critical":
    case "error":
      return t.criticalEvents;
    case "warning":
      return t.warningEvents;
    default:
      return t.normalEvents;
  }
}

/** 筛选下拉的选项值 */
export const EVENT_SEVERITY_FILTERS: EventSeverity[] = ["critical", "warning", "info"];

/** 事件是否符合严重程度筛选（选 critical 时 error 也算） */
export function matchesSeverityFilter(severity: EventSeverity, filter: string): boolean {
  if (!filter) return true;
  if (filter === "critical") return severity === "critical" || severity === "error";
  return severity === filter;
}
