"use client";

import { Activity, CheckCircle, AlertTriangle } from "lucide-react";
import { StatusBadge } from "@/components/common";
import type { StatefulSetDetail } from "@/api/workload";

interface OverviewTabProps {
  detail: StatefulSetDetail;
}

export function OverviewTab({ detail }: OverviewTabProps) {
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
        <h3 className="text-sm font-semibold text-default mb-3">副本状态</h3>
        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          {[
            { label: "期望", value: detail.replicas, color: "text-default" },
            { label: "就绪", value: detail.ready, color: "text-green-500" },
            { label: "当前", value: detail.current, color: "text-blue-500" },
            { label: "已更新", value: detail.updated, color: "text-purple-500" },
            { label: "可用", value: detail.available, color: "text-cyan-500" },
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
        <h3 className="text-sm font-semibold text-default mb-3">基本信息</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: "名称", value: detail.name },
            { label: "命名空间", value: detail.namespace },
            { label: "Headless Service", value: detail.serviceName || "-" },
            { label: "Age", value: detail.age || "-" },
            { label: "Pod 管理策略", value: detail.spec.podManagementPolicy || "OrderedReady" },
            { label: "创建时间", value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
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
              <div className="text-xs text-muted mb-1">当前 Revision</div>
              <div className="text-sm font-mono text-default truncate">
                {detail.status.currentRevision || "-"}
              </div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">更新 Revision</div>
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
