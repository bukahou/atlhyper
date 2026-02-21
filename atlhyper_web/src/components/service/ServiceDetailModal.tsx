"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getServiceDetail } from "@/datasource/cluster";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import type { ServiceDetail, ServicePort, BackendEndpoint } from "@/types/cluster";
import {
  Globe,
  Network,
  Server,
  Tag,
  CheckCircle,
  XCircle,
} from "lucide-react";

interface ServiceDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  serviceName: string;
}

type TabType = "overview" | "ports" | "endpoints" | "selector";

export function ServiceDetailModal({
  isOpen,
  onClose,
  namespace,
  serviceName,
}: ServiceDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<ServiceDetail | null>(null);
  const { t } = useI18n();

  const fetchDetail = useCallback(async () => {
    if (!serviceName || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getServiceDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Name: serviceName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.service.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [namespace, serviceName, t.service.loadFailed]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.service.overview, icon: <Globe className="w-4 h-4" /> },
    { key: "ports", label: t.service.ports, icon: <Network className="w-4 h-4" /> },
    { key: "endpoints", label: t.service.endpoints, icon: <Server className="w-4 h-4" /> },
    { key: "selector", label: t.service.selector, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen={isOpen} onClose={onClose} title={`Service: ${serviceName}`} size="xl">
      {loading ? (
        <div className="py-12">
          <LoadingSpinner />
        </div>
      ) : error ? (
        <div className="p-6 text-center text-red-500">{error}</div>
      ) : detail ? (
        <div className="flex flex-col h-full">
          {/* Tabs */}
          <div className="flex border-b border-[var(--border-color)] px-4 shrink-0">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === tab.key
                    ? "border-primary text-primary"
                    : "border-transparent text-muted hover:text-default"
                }`}
              >
                {tab.icon}
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab Content */}
          <div className="flex-1 overflow-auto p-6">
            {activeTab === "overview" && <OverviewTab detail={detail} t={t} />}
            {activeTab === "ports" && <PortsTab ports={detail.ports || []} t={t} />}
            {activeTab === "endpoints" && <EndpointsTab backends={detail.backends} t={t} />}
            {activeTab === "selector" && <SelectorTab selector={detail.selector || {}} t={t} />}
          </div>
        </div>
      ) : null}
    </Drawer>
  );
}

// 概览 Tab
function OverviewTab({ detail, t }: { detail: ServiceDetail; t: ReturnType<typeof useI18n>["t"] }) {
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
      {/* 基本信息 */}
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

// 端口 Tab
function PortsTab({ ports, t }: { ports: ServicePort[]; t: ReturnType<typeof useI18n>["t"] }) {
  if (ports.length === 0) {
    return <div className="text-center py-8 text-muted">{t.service.noPorts}</div>;
  }

  return (
    <div className="space-y-3">
      {ports.map((port, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <div className="flex items-center gap-2">
              <h4 className="font-medium text-default">{port.name || `Port ${i + 1}`}</h4>
              <StatusBadge status={port.protocol} type="info" />
            </div>
            {port.appProtocol && (
              <span className="text-xs text-muted">App: {port.appProtocol}</span>
            )}
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <div className="text-xs text-muted mb-1">{t.service.port}</div>
              <div className="text-sm font-mono text-default">{port.port}</div>
            </div>
            <div>
              <div className="text-xs text-muted mb-1">{t.service.targetPort}</div>
              <div className="text-sm font-mono text-default">{port.targetPort}</div>
            </div>
            {port.nodePort && port.nodePort > 0 && (
              <div>
                <div className="text-xs text-muted mb-1">{t.service.nodePort}</div>
                <div className="text-sm font-mono text-default">{port.nodePort}</div>
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

// 端点 Tab
function EndpointsTab({ backends, t }: { backends?: { ready: number; notReady: number; total: number; endpoints?: BackendEndpoint[] }; t: ReturnType<typeof useI18n>["t"] }) {
  if (!backends || !backends.endpoints || backends.endpoints.length === 0) {
    return <div className="text-center py-8 text-muted">{t.service.noEndpoints}</div>;
  }

  return (
    <div className="space-y-3">
      {backends.endpoints.map((ep, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              {ep.ready ? (
                <CheckCircle className="w-4 h-4 text-green-500" />
              ) : (
                <XCircle className="w-4 h-4 text-red-500" />
              )}
              <span className="font-mono text-default">{ep.address}</span>
            </div>
            <StatusBadge status={ep.ready ? t.service.ready : t.service.notReady} type={ep.ready ? "success" : "error"} />
          </div>

          <div className="grid grid-cols-2 md:grid-cols-3 gap-3 text-sm">
            {ep.nodeName && (
              <div>
                <span className="text-muted">Node: </span>
                <span className="text-default">{ep.nodeName}</span>
              </div>
            )}
            {ep.zone && (
              <div>
                <span className="text-muted">Zone: </span>
                <span className="text-default">{ep.zone}</span>
              </div>
            )}
            {ep.targetRef && (
              <div>
                <span className="text-muted">{ep.targetRef.kind}: </span>
                <span className="text-default">{ep.targetRef.name}</span>
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

// 选择器 Tab
function SelectorTab({ selector, t }: { selector: Record<string, string>; t: ReturnType<typeof useI18n>["t"] }) {
  const entries = Object.entries(selector);

  if (entries.length === 0) {
    return <div className="text-center py-8 text-muted">{t.service.noSelector}</div>;
  }

  return (
    <div className="space-y-2">
      {entries.map(([key, value]) => (
        <div key={key} className="bg-[var(--background)] rounded-lg p-3 flex items-start gap-2">
          <span className="text-sm font-mono text-primary break-all">{key}</span>
          <span className="text-muted">=</span>
          <span className="text-sm font-mono text-default break-all">{value || '""'}</span>
        </div>
      ))}
    </div>
  );
}
