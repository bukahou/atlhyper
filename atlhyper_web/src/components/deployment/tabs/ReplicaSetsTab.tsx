"use client";

import { Calendar } from "lucide-react";
import type { DeploymentReplicaSet } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";

interface ReplicaSetsTabProps {
  replicaSets: DeploymentReplicaSet[];
  t: ReturnType<typeof useI18n>["t"];
}

export function ReplicaSetsTab({ replicaSets, t }: ReplicaSetsTabProps) {
  if (replicaSets.length === 0) {
    return <div className="text-center py-8 text-muted">{t.deployment.noReplicaSets}</div>;
  }

  // 按 revision 排序，最新的在前
  const sorted = [...replicaSets].sort((a, b) => {
    const ra = parseInt(a.revision || "0", 10);
    const rb = parseInt(b.revision || "0", 10);
    return rb - ra;
  });

  return (
    <div className="space-y-3">
      {sorted.map((rs, i) => (
        <div
          key={rs.name}
          className={`bg-[var(--background)] rounded-lg p-4 ${i === 0 ? "ring-2 ring-primary" : ""}`}
        >
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <span className="font-medium text-default">{rs.name}</span>
              {rs.revision && (
                <span className="px-2 py-0.5 bg-card text-xs rounded">Rev {rs.revision}</span>
              )}
              {i === 0 && (
                <span className="px-2 py-0.5 bg-primary/10 text-primary text-xs rounded">{t.deployment.current}</span>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Calendar className="w-4 h-4 text-muted" />
              <span className="text-sm text-muted">{rs.createdAt ? new Date(rs.createdAt).toLocaleString() : "-"}</span>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <div className="text-lg font-bold text-default">{rs.replicas}</div>
              <div className="text-xs text-muted">{t.deployment.desired}</div>
            </div>
            <div>
              <div className="text-lg font-bold text-green-500">{rs.ready}</div>
              <div className="text-xs text-muted">{t.deployment.ready}</div>
            </div>
            <div>
              <div className="text-lg font-bold text-blue-500">{rs.available}</div>
              <div className="text-xs text-muted">{t.deployment.available}</div>
            </div>
          </div>
          {rs.image && (
            <div className="mt-2 text-xs font-mono text-muted truncate">{rs.image}</div>
          )}
        </div>
      ))}
    </div>
  );
}
