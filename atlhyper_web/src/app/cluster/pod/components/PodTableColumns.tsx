"use client";

import { StatusBadge, type TableColumn } from "@/components/common";
import { RotateCcw, Eye } from "lucide-react";
import type { PodItem } from "@/types/cluster";
import type { Translations } from "@/types/i18n";

export function getPodColumns(
  t: Translations,
  onViewDetail: (pod: PodItem) => void,
  onRestart: (pod: PodItem) => void,
): TableColumn<PodItem>[] {
  return [
    {
      key: "name",
      header: t.common.name,
      mobileTitle: true,
      render: (pod) => (
        <div>
          <span className="font-medium text-default">{pod.name || "-"}</span>
          <div className="text-xs text-muted">{pod.deployment || "-"}</div>
        </div>
      ),
    },
    { key: "namespace", header: t.common.namespace },
    {
      key: "phase",
      header: t.common.status,
      render: (pod) => <StatusBadge status={pod.phase || "Unknown"} />,
    },
    {
      key: "ready",
      header: "Ready",
      render: (pod) => <span className="font-mono text-sm">{pod.ready || "-"}</span>,
    },
    { key: "node", header: "Node", mobileVisible: false },
    {
      key: "cpu",
      header: "CPU",
      mobileVisible: false,
      render: (pod) => (
        <span className="text-sm">{pod.cpuText || "-"}</span>
      ),
    },
    {
      key: "memory",
      header: "Memory",
      mobileVisible: false,
      render: (pod) => (
        <span className="text-sm">{pod.memoryText || "-"}</span>
      ),
    },
    {
      key: "restarts",
      header: "Restarts",
      render: (pod) => <span>{pod.restarts ?? 0}</span>,
    },
    {
      key: "age",
      header: "Age",
      mobileVisible: false,
      render: (pod) => <span className="text-sm text-muted">{pod.age || "-"}</span>,
    },
    {
      key: "action",
      header: t.common.action,
      mobileVisible: false,
      render: (pod) => (
        <div className="flex items-center gap-1">
          <button
            onClick={(e) => {
              e.stopPropagation();
              onViewDetail(pod);
            }}
            className="p-2 hover-bg rounded-lg"
            title={t.pod.viewDetails}
          >
            <Eye className="w-4 h-4 text-muted" />
          </button>
          <button
            onClick={(e) => {
              e.stopPropagation();
              onRestart(pod);
            }}
            className="p-2 hover-bg rounded-lg"
            title={t.pod.restart}
          >
            <RotateCcw className="w-4 h-4 text-muted" />
          </button>
        </div>
      ),
    },
  ];
}
