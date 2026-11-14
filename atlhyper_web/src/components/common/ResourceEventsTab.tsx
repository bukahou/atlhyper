"use client";

import { useQuery } from "@tanstack/react-query";
import { Loader2, AlertTriangle, Info, XCircle } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { resourceEventsQuery } from "@/queries/cluster";
import type { EventLog } from "@/types/cluster";

// ──────────────────────────────────────────────────────────────
// 资源事件时间线（G4 接线，2026-08-29 能力盘点）
// ──────────────────────────────────────────────────────────────
//
// 后端 /events/by-resource 一直完整，此前无数据未接线；压测期间
// 副本增删产生了真实事件流，接入资源详情弹窗作为独立 Tab。
// 通用组件：Pod / Deployment 等任何资源详情都可复用。

interface ResourceEventsTabProps {
  clusterId: string;
  kind: string;
  namespace?: string;
  name: string;
}

function severityIcon(sev: string) {
  const s = (sev || "").toLowerCase();
  if (s === "error") return <XCircle className="w-3.5 h-3.5 text-red-500 shrink-0" />;
  if (s === "warning") return <AlertTriangle className="w-3.5 h-3.5 text-yellow-500 shrink-0" />;
  return <Info className="w-3.5 h-3.5 text-blue-400 shrink-0" />;
}

// eventTime 可能为空（老事件只有 time），两者都空显示占位
function eventTimeText(e: EventLog): string {
  const raw = e.eventTime || e.time;
  if (!raw) return "—";
  const d = new Date(raw);
  return isNaN(d.getTime()) ? "—" : d.toLocaleString();
}

export function ResourceEventsTab({ clusterId, kind, namespace, name }: ResourceEventsTabProps) {
  const { t } = useI18n();
  // 键里带 kind / namespace / name：换一个资源就是另一份数据，不会把上一个资源的事件显示出来
  const query = useQuery(resourceEventsQuery(clusterId, { kind, namespace, name }));
  const events = query.data;

  if (events === undefined) {
    if (query.isError) {
      return <div className="py-8 text-center text-sm text-muted">{t.common.loadFailed}</div>;
    }
    return (
      <div className="flex justify-center py-8">
        <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
      </div>
    );
  }
  if (events.length === 0) {
    return <div className="py-8 text-center text-sm text-muted">{t.event.noEvents}</div>;
  }

  return (
    <div className="space-y-2">
      {events.map((e, i) => (
        <div key={i} className="flex items-start gap-3 bg-[var(--background)] rounded-lg p-3">
          {severityIcon(e.severity)}
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline gap-2 flex-wrap">
              <span className="text-xs font-medium text-default">{e.reason}</span>
              <span className="text-[10px] text-muted font-mono">{eventTimeText(e)}</span>
            </div>
            <p className="text-xs text-muted mt-1 break-words">{e.message}</p>
          </div>
        </div>
      ))}
    </div>
  );
}
