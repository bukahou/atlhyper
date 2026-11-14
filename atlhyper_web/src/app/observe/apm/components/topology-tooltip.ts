/**
 * 拓扑图悬停提示的 HTML（节点与边两种）
 *
 * G6 的 tooltip 只收 HTML 字符串，所以一律经 safeHtml 拼接：服务名等来自采集数据，不能当 HTML。
 */
import { formatDurationMs } from "@/lib/format";
import { safeHtml } from "@/lib/safe-html";
import type { ApmTranslations } from "@/types/i18n";
import { ringColor } from "./topology-utils";

/** G6 交给 tooltip 的图元素（节点或边）里用到的部分 */
export interface TopologyTooltipItem {
  id: string;
  data?: {
    rps?: number;
    successRate?: number;
    p99Ms?: number;
    namespace?: string;
    type?: string;
    callCount?: number;
    avgMs?: number;
    errorRate?: number;
  };
}

export function buildTopologyTooltip(items: TopologyTooltipItem[] | undefined, t: ApmTranslations): string {
  if (!items?.length) return "";
  const item = items[0];
  const d = item.data;
  if (!d) return safeHtml`${item.id}`.toString();

  // Node tooltip
  if (d.rps !== undefined) {
    const color = ringColor(
      d.successRate !== undefined && d.successRate < 0.95 ? "critical" : "healthy",
      d.successRate ?? 1
    );
    return safeHtml`<div style="padding:8px 12px;font-size:12px;border-radius:8px;background:rgba(0,0,0,0.88);color:#fff;min-width:140px;border-left:3px solid ${color}">
      <div style="font-weight:600;margin-bottom:4px">${item.id}</div>
      ${d.namespace ? safeHtml`<div style="opacity:0.6;margin-bottom:2px">${d.namespace}</div>` : ""}
      <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
        <span>RPS</span><span>${d.rps?.toFixed(3)}</span>
      </div>
      <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
        <span>${t.successRate}</span><span>${((d.successRate ?? 1) * 100).toFixed(1)}%</span>
      </div>
      <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
        <span>P99</span><span>${formatDurationMs(d.p99Ms ?? 0)}</span>
      </div>
    </div>`.toString();
  }

  // Edge tooltip
  if (d.callCount !== undefined) {
    return safeHtml`<div style="padding:8px 12px;font-size:12px;border-radius:8px;background:rgba(0,0,0,0.88);color:#fff;min-width:120px">
      <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
        <span>${t.topoCalls}</span><span>${d.callCount}</span>
      </div>
      <div style="display:flex;justify-content:space-between;gap:12px;opacity:0.8">
        <span>${t.topoLatency}</span><span>${formatDurationMs(d.avgMs ?? 0)}</span>
      </div>
      ${(d.errorRate ?? 0) > 0 ? safeHtml`<div style="display:flex;justify-content:space-between;gap:12px;color:#f87171">
        <span>${t.topoErrorRate}</span><span>${((d.errorRate ?? 0) * 100).toFixed(1)}%</span>
      </div>` : ""}
    </div>`.toString();
  }

  return safeHtml`${item.id}`.toString();
}
