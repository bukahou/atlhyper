"use client";

import { StatusBadge } from "@/components/common";
import type { NamespaceDetail } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";

interface OverviewTabProps {
  detail: NamespaceDetail;
  t: ReturnType<typeof useI18n>["t"];
}

export function OverviewTab({ detail, t }: OverviewTabProps) {
  const basicInfo = [
    { label: t.common.name, value: detail.name },
    { label: t.common.status, value: <StatusBadge status={detail.phase} /> },
    { label: t.namespace.age, value: detail.age || "-" },
    { label: t.common.createdAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
    { label: t.namespace.labels, value: detail.labelCount },
    { label: t.namespace.annotations, value: detail.annotationCount },
  ];

  const workloads = [
    { label: "Pods", value: detail.pods, running: detail.podsRunning },
    { label: "Deployments", value: detail.deployments },
    { label: "StatefulSets", value: detail.statefulSets },
    { label: "DaemonSets", value: detail.daemonSets },
    { label: "Jobs", value: detail.jobs },
    { label: "CronJobs", value: detail.cronJobs },
  ];

  const network = [
    { label: "Services", value: detail.services },
    { label: "Ingresses", value: detail.ingresses },
    { label: "NetworkPolicies", value: detail.networkPolicies },
  ];

  const config = [
    { label: "ConfigMaps", value: detail.configMaps },
    { label: "Secrets", value: detail.secrets },
    { label: "ServiceAccounts", value: detail.serviceAccounts },
    { label: "PVCs", value: detail.persistentVolumeClaims },
  ];

  return (
    <div className="space-y-6">
      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.namespace.basicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {basicInfo.map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Pod 状态 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.namespace.podStatus}</h3>
        <div className="grid grid-cols-4 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-default">{detail.pods}</div>
            <div className="text-xs text-muted mt-1">{t.namespace.total}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-green-500">{detail.podsRunning}</div>
            <div className="text-xs text-muted mt-1">Running</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-yellow-500">{detail.podsPending}</div>
            <div className="text-xs text-muted mt-1">Pending</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-red-500">{detail.podsFailed}</div>
            <div className="text-xs text-muted mt-1">Failed</div>
          </div>
        </div>
      </div>

      {/* 工作负载 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.namespace.workloads}</h3>
        <div className="grid grid-cols-3 md:grid-cols-6 gap-3">
          {workloads.map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3 text-center">
              <div className="text-xl font-bold text-default">{item.value}</div>
              <div className="text-xs text-muted mt-1">{item.label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 网络 & 配置 */}
      <div className="grid grid-cols-2 gap-6">
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.namespace.network}</h3>
          <div className="grid grid-cols-3 gap-3">
            {network.map((item, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-3 text-center">
                <div className="text-xl font-bold text-default">{item.value}</div>
                <div className="text-xs text-muted mt-1">{item.label}</div>
              </div>
            ))}
          </div>
        </div>
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.namespace.config}</h3>
          <div className="grid grid-cols-2 gap-3">
            {config.map((item, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-3 text-center">
                <div className="text-xl font-bold text-default">{item.value}</div>
                <div className="text-xs text-muted mt-1">{item.label}</div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* 指标 */}
      {detail.metrics && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.namespace.resourceUsage}</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-4">
              <div className="text-sm font-medium mb-2">CPU</div>
              <div className="text-lg font-bold text-default">{detail.metrics.cpu.usage}</div>
              {detail.metrics.cpu.utilPct !== undefined && (
                <div className="text-xs text-muted mt-1">{detail.metrics.cpu.utilPct.toFixed(1)}% {t.namespace.utilization}</div>
              )}
            </div>
            <div className="bg-[var(--background)] rounded-lg p-4">
              <div className="text-sm font-medium mb-2">Memory</div>
              <div className="text-lg font-bold text-default">{detail.metrics.memory.usage}</div>
              {detail.metrics.memory.utilPct !== undefined && (
                <div className="text-xs text-muted mt-1">{detail.metrics.memory.utilPct.toFixed(1)}% {t.namespace.utilization}</div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
