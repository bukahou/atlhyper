"use client";

import { Activity, CheckCircle, AlertTriangle } from "lucide-react";
import { StatusBadge } from "@/components/common";
import type { StatefulSetDetail } from "@/api/workload";
import type { useI18n } from "@/i18n/context";

type Translations = ReturnType<typeof useI18n>["t"];

interface OverviewTabProps {
  detail: StatefulSetDetail;
  t: Translations;
}

export function OverviewTab({ detail, t }: OverviewTabProps) {
  const getRolloutBadgeType = (phase: string): "success" | "warning" | "error" | "info" => {
    switch (phase) {
      case "Complete": return "success";
      case "Progressing": return "info";
      case "Degraded": return "error";
      default: return "warning";
    }
  };

  return (
    <div className="space-y-6">
      {/* Rollout Status */}
      {detail.rollout && (
        <div className="flex items-center gap-3 p-4 bg-[var(--background)] rounded-lg">
          <Activity className="w-5 h-5 text-primary" />
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <StatusBadge status={detail.rollout.phase} type={getRolloutBadgeType(detail.rollout.phase)} />
              {detail.rollout.badges?.map((badge, i) => (
                <StatusBadge key={i} status={badge} type="info" />
              ))}
            </div>
            {detail.rollout.message && (
              <p className="text-sm text-muted mt-1">{detail.rollout.message}</p>
            )}
          </div>
        </div>
      )}

      {/* 副本状态 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.statefulset.replicaStatus}</h3>
        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          {[
            { label: t.statefulset.desired, value: detail.replicas, color: "text-default" },
            { label: t.statefulset.ready, value: detail.ready, color: "text-green-500" },
            { label: t.statefulset.current, value: detail.current, color: "text-blue-500" },
            { label: t.statefulset.updated, value: detail.updated, color: "text-purple-500" },
            { label: t.statefulset.available, value: detail.available, color: "text-cyan-500" },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3 text-center">
              <div className={`text-2xl font-bold ${item.color}`}>{item.value}</div>
              <div className="text-xs text-muted mt-1">{item.label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.statefulset.basicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: t.common.name, value: detail.name },
            { label: t.common.namespace, value: detail.namespace },
            { label: "Headless Service", value: detail.serviceName || "-" },
            { label: "Age", value: detail.age || "-" },
            { label: t.statefulset.podManagementPolicy, value: detail.spec.podManagementPolicy || "OrderedReady" },
            { label: t.common.createdAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium truncate">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Revision 信息 */}
      {(detail.status.currentRevision || detail.status.updateRevision) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Revision</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{t.statefulset.currentRevision}</div>
              <div className="text-sm font-mono text-default truncate">
                {detail.status.currentRevision || "-"}
              </div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{t.statefulset.updateRevision}</div>
              <div className="text-sm font-mono text-default truncate">
                {detail.status.updateRevision || "-"}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Conditions */}
      {detail.conditions && detail.conditions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Conditions</h3>
          <div className="space-y-2">
            {detail.conditions.map((c, i) => (
              <div key={i} className="flex items-start gap-3 p-3 bg-[var(--background)] rounded-lg">
                {c.status === "True" ? (
                  <CheckCircle className="w-4 h-4 text-green-500 mt-0.5" />
                ) : (
                  <AlertTriangle className="w-4 h-4 text-yellow-500 mt-0.5" />
                )}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-default">{c.type}</span>
                    <StatusBadge status={c.status} type={c.status === "True" ? "success" : "warning"} />
                  </div>
                  {c.message && (
                    <p className="text-sm text-muted mt-1 break-words">{c.message}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
