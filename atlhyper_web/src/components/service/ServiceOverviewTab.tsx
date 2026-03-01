"use client";

import { StatusBadge } from "@/components/common";
import type { useI18n } from "@/i18n/context";
import type { ServiceDetail } from "@/types/cluster";

interface ServiceOverviewTabProps {
  detail: ServiceDetail;
  t: ReturnType<typeof useI18n>["t"];
}

export function ServiceOverviewTab({ detail, t }: ServiceOverviewTabProps) {
  const getTypeStatus = (type: string): "success" | "info" | "default" => {
    if (type === "LoadBalancer") return "success";
    if (type === "NodePort") return "info";
    return "default";
  };

  const infoItems = [
    { label: t.common.name, value: detail.name },
    { label: t.common.namespace, value: detail.namespace },
    { label: t.service.serviceType, value: <StatusBadge status={detail.type} type={getTypeStatus(detail.type)} /> },
    { label: t.service.age, value: detail.age || "-" },
    { label: t.common.createdAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
    { label: t.service.sessionAffinity, value: detail.sessionAffinity || "None" },
  ];

  return (
    <div className="space-y-6">
      {/* Basic info */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.service.basicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {infoItems.map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Cluster IPs */}
      {detail.clusterIPs && detail.clusterIPs.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Cluster IPs</h3>
          <div className="flex flex-wrap gap-2">
            {detail.clusterIPs.map((ip, i) => (
              <span key={i} className="px-3 py-1.5 bg-[var(--background)] text-sm font-mono rounded">
                {ip}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* External IPs */}
      {detail.externalIPs && detail.externalIPs.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.service.externalIP}</h3>
          <div className="flex flex-wrap gap-2">
            {detail.externalIPs.map((ip, i) => (
              <span key={i} className="px-3 py-1.5 bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 text-sm font-mono rounded">
                {ip}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* LoadBalancer Ingress */}
      {detail.loadBalancerIngress && detail.loadBalancerIngress.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.service.loadBalancerIP}</h3>
          <div className="flex flex-wrap gap-2">
            {detail.loadBalancerIngress.map((addr, i) => (
              <span key={i} className="px-3 py-1.5 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-sm font-mono rounded">
                {addr}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Traffic Policies */}
      {(detail.externalTrafficPolicy || detail.internalTrafficPolicy) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.service.trafficPolicy}</h3>
          <div className="grid grid-cols-2 gap-4">
            {detail.externalTrafficPolicy && (
              <div className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-xs text-muted mb-1">External Traffic</div>
                <div className="text-sm text-default font-medium">{detail.externalTrafficPolicy}</div>
              </div>
            )}
            {detail.internalTrafficPolicy && (
              <div className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-xs text-muted mb-1">Internal Traffic</div>
                <div className="text-sm text-default font-medium">{detail.internalTrafficPolicy}</div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* IP Families */}
      {detail.ipFamilies && detail.ipFamilies.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">IP Families</h3>
          <div className="flex flex-wrap gap-2">
            {detail.ipFamilies.map((family, i) => (
              <StatusBadge key={i} status={family} type="info" />
            ))}
            {detail.ipFamilyPolicy && (
              <span className="text-sm text-muted">({detail.ipFamilyPolicy})</span>
            )}
          </div>
        </div>
      )}

      {/* Backends Summary */}
      {detail.backends && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.service.endpointStatus}</h3>
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-4 text-center">
              <div className="text-2xl font-bold text-green-500">{detail.backends.ready}</div>
              <div className="text-xs text-muted mt-1">{t.service.ready}</div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-4 text-center">
              <div className="text-2xl font-bold text-red-500">{detail.backends.notReady}</div>
              <div className="text-xs text-muted mt-1">{t.service.notReady}</div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-4 text-center">
              <div className="text-2xl font-bold text-default">{detail.backends.total}</div>
              <div className="text-xs text-muted mt-1">{t.service.total}</div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
