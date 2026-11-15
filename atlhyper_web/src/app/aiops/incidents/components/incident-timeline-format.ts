/**
 * 事件详情：时间线展示形态（timelineView）的文案映射（纯函数）
 *
 * 后端已经把检测记录合并成 kind=anomaly 的条目、给出状态与严重度的变化；
 * 这里只负责「kind / 严重度码 → i18n 文案」，不出现 healthy / warning / critical 等内部写法。
 */
import type { IncidentDetailTranslations } from "@/types/i18n-incident";
import type { IncidentTimelineItem } from "@/types/incident-view";
import { formatShortTime } from "./incident-format";
import { fillTemplate } from "./incident-view-format";

/** 折叠时的摘要：几个变化（非检测条目）、合并了几条检测 */
export function timelineSummary(items: IncidentTimelineItem[]): { changes: number; merged: number } {
  let changes = 0;
  let merged = 0;
  for (const item of items) {
    if (item.kind === "anomaly") merged += item.count ?? 0;
    else changes += 1;
  }
  return { changes, merged };
}

/** 一条时间线的显示：主文案 + 补充说明（风险、严重度、合并范围） */
export interface TimelineLine {
  label: string;
  notes: string[];
  /** 合并的检测记录：显示得淡一些 */
  merged: boolean;
}

/** 严重度码 → 显示名；没见过的码原样显示，不吞掉 */
type SeverityLabels = Record<string, string>;

export function timelineLine(
  item: IncidentTimelineItem,
  texts: IncidentDetailTranslations,
  severityLabels: SeverityLabels,
): TimelineLine {
  const severity = item.severity ? (severityLabels[item.severity] ?? item.severity) : "";
  const kindLabels: Record<string, string> = texts.timelineKind;

  if (item.kind === "anomaly") {
    const first = item.firstAt ?? item.at;
    const last = item.lastAt ?? item.at;
    return {
      label: texts.timelineKind.anomaly,
      notes: [
        fillTemplate(texts.timelineAnomalyMerged, {
          count: item.count ?? 0,
          first: formatShortTime(first),
          last: formatShortTime(last),
        }),
      ],
      merged: true,
    };
  }

  const notes: string[] = [];
  let label = kindLabels[item.kind] ?? texts.timelineUnknownKind;
  if (item.kind === "severity_raised" && severity !== "") {
    // 「严重度升高」带上升到哪一档，严重度不再单独列
    label = fillTemplate(texts.timelineSeverityRaisedTo, { severity });
  } else if (severity !== "") {
    notes.push(fillTemplate(texts.timelineSeverity, { severity }));
  }
  if (item.risk !== undefined && item.risk !== null) {
    notes.push(fillTemplate(texts.timelineRisk, { risk: Math.round(item.risk) }));
  }
  return { label, notes, merged: false };
}
