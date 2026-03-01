"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getServiceDetail } from "@/datasource/cluster";
import { useClusterStore } from "@/store/clusterStore";
import { useI18n } from "@/i18n/context";
import type { ServiceDetail, ServicePort, BackendEndpoint } from "@/types/cluster";
import { ServiceOverviewTab } from "./ServiceOverviewTab";
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
  const { currentClusterId } = useClusterStore();

  const fetchDetail = useCallback(async () => {
    if (!serviceName || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getServiceDetail({
        ClusterID: currentClusterId,
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
            {activeTab === "overview" && <ServiceOverviewTab detail={detail} t={t} />}
            {activeTab === "ports" && <PortsTab ports={detail.ports || []} t={t} />}
            {activeTab === "endpoints" && <EndpointsTab backends={detail.backends} t={t} />}
            {activeTab === "selector" && <SelectorTab selector={detail.selector || {}} t={t} />}
          </div>
        </div>
      ) : null}
    </Drawer>
  );
}

// Ports Tab
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

// Endpoints Tab
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

// Selector Tab
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
