"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getIngressDetail } from "@/api/ingress";
import { getCurrentClusterId } from "@/config/cluster";
import type {
  IngressDetail,
  IngressRuleDTO,
  IngressTLSDTO,
} from "@/types/cluster";
import {
  Globe,
  Route,
  Lock,
  Tag,
  Server,
  ArrowRight,
} from "lucide-react";

interface IngressDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  ingressName: string;
}

type TabType = "overview" | "rules" | "tls" | "annotations";

export function IngressDetailModal({
  isOpen,
  onClose,
  namespace,
  ingressName,
}: IngressDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<IngressDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!ingressName || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getIngressDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Name: ingressName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [namespace, ingressName]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: "概览", icon: <Globe className="w-4 h-4" /> },
    { key: "rules", label: "路由规则", icon: <Route className="w-4 h-4" /> },
    { key: "tls", label: "TLS", icon: <Lock className="w-4 h-4" /> },
    { key: "annotations", label: "注解", icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Ingress: ${ingressName}`} size="xl">
      {loading ? (
        <div className="py-12">
          <LoadingSpinner />
        </div>
      ) : error ? (
        <div className="p-6 text-center text-red-500">{error}</div>
      ) : detail ? (
        <div className="flex flex-col h-[70vh]">
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
            {activeTab === "overview" && <OverviewTab detail={detail} />}
            {activeTab === "rules" && <RulesTab rules={detail.spec?.rules || []} defaultBackend={detail.spec?.defaultBackend} />}
            {activeTab === "tls" && <TLSTab tls={detail.spec?.tls || []} />}
            {activeTab === "annotations" && <AnnotationsTab annotations={detail.annotations || {}} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

// 概览 Tab
function OverviewTab({ detail }: { detail: IngressDetail }) {
  const infoItems = [
    { label: "名称", value: detail.name },
    { label: "命名空间", value: detail.namespace },
    { label: "Ingress Class", value: detail.class || detail.spec?.ingressClassName || "-" },
    { label: "Controller", value: detail.controller || "-" },
    { label: "Age", value: detail.age || "-" },
    { label: "创建时间", value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
  ];

  return (
    <div className="space-y-6">
      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">基本信息</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {infoItems.map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* TLS 状态 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">TLS 状态</h3>
        <div className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center gap-3">
            {detail.tlsEnabled ? (
              <>
                <Lock className="w-5 h-5 text-green-500" />
                <span className="text-green-600 font-medium">TLS 已启用</span>
                <StatusBadge status="Enabled" type="success" />
              </>
            ) : (
              <>
                <Lock className="w-5 h-5 text-muted" />
                <span className="text-muted">TLS 未启用</span>
                <StatusBadge status="Disabled" type="default" />
              </>
            )}
          </div>
        </div>
      </div>

      {/* Hosts */}
      {detail.hosts && detail.hosts.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Hosts</h3>
          <div className="flex flex-wrap gap-2">
            {detail.hosts.map((host, i) => (
              <span key={i} className="px-3 py-1.5 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-sm font-mono rounded">
                {host}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* LoadBalancer */}
      {detail.loadBalancer && detail.loadBalancer.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">LoadBalancer IPs</h3>
          <div className="flex flex-wrap gap-2">
            {detail.loadBalancer.map((ip, i) => (
              <span key={i} className="px-3 py-1.5 bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 text-sm font-mono rounded">
                {ip}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* 规则统计 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">规则统计</h3>
        <div className="grid grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-primary">{detail.spec?.rules?.length || 0}</div>
            <div className="text-xs text-muted mt-1">路由规则</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-blue-500">
              {detail.spec?.rules?.reduce((sum, r) => sum + (r.paths?.length || 0), 0) || 0}
            </div>
            <div className="text-xs text-muted mt-1">路径数</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-green-500">{detail.spec?.tls?.length || 0}</div>
            <div className="text-xs text-muted mt-1">TLS 证书</div>
          </div>
        </div>
      </div>
    </div>
  );
}

// 路由规则 Tab
function RulesTab({ rules, defaultBackend }: { rules: IngressRuleDTO[]; defaultBackend?: IngressDetail["spec"]["defaultBackend"] }) {
  if (rules.length === 0 && !defaultBackend) {
    return <div className="text-center py-8 text-muted">暂无路由规则</div>;
  }

  return (
    <div className="space-y-4">
      {/* Default Backend */}
      {defaultBackend && (
        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
          <h4 className="font-medium text-yellow-700 dark:text-yellow-400 mb-2">默认后端</h4>
          <BackendDisplay backend={defaultBackend} />
        </div>
      )}

      {/* Rules */}
      {rules.map((rule, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center gap-2 mb-4">
            <Globe className="w-4 h-4 text-primary" />
            <h4 className="font-medium text-default">{rule.host || "*"}</h4>
          </div>

          <div className="space-y-3">
            {rule.paths.map((path, j) => (
              <div key={j} className="flex items-center gap-3 p-3 bg-card rounded-lg border border-[var(--border-color)]">
                <div className="flex-1 flex items-center gap-3">
                  <span className="px-2 py-1 text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded">
                    {path.pathType || "Prefix"}
                  </span>
                  <span className="font-mono text-sm text-default">{path.path || "/"}</span>
                </div>
                <ArrowRight className="w-4 h-4 text-muted" />
                <div className="flex items-center gap-2">
                  <Server className="w-4 h-4 text-muted" />
                  <BackendDisplay backend={path.backend} />
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

// Backend 显示组件
function BackendDisplay({ backend }: { backend: IngressDetail["spec"]["defaultBackend"] }) {
  if (!backend) return <span className="text-muted">-</span>;

  if (backend.type === "Service" && backend.service) {
    const port = backend.service.portName || backend.service.portNumber;
    return (
      <span className="font-mono text-sm text-default">
        {backend.service.name}{port ? `:${port}` : ""}
      </span>
    );
  }

  if (backend.type === "Resource" && backend.resource) {
    return (
      <span className="font-mono text-sm text-default">
        {backend.resource.kind}/{backend.resource.name}
      </span>
    );
  }

  return <span className="text-muted">-</span>;
}

// TLS Tab
function TLSTab({ tls }: { tls: IngressTLSDTO[] }) {
  if (tls.length === 0) {
    return <div className="text-center py-8 text-muted">暂无 TLS 配置</div>;
  }

  return (
    <div className="space-y-3">
      {tls.map((item, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center gap-2 mb-3">
            <Lock className="w-4 h-4 text-green-500" />
            <h4 className="font-medium text-default">{item.secretName}</h4>
          </div>

          {item.hosts && item.hosts.length > 0 && (
            <div>
              <div className="text-xs text-muted mb-2">Hosts:</div>
              <div className="flex flex-wrap gap-2">
                {item.hosts.map((host, j) => (
                  <span key={j} className="px-2 py-1 text-sm font-mono bg-card border border-[var(--border-color)] rounded">
                    {host}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// 注解 Tab
function AnnotationsTab({ annotations }: { annotations: Record<string, string> }) {
  const entries = Object.entries(annotations);

  if (entries.length === 0) {
    return <div className="text-center py-8 text-muted">暂无注解</div>;
  }

  return (
    <div className="space-y-2">
      {entries.map(([key, value]) => (
        <div key={key} className="bg-[var(--background)] rounded-lg p-3">
          <div className="text-sm font-mono text-primary break-all mb-1">{key}</div>
          <div className="text-sm font-mono text-default break-all">{value || '""'}</div>
        </div>
      ))}
    </div>
  );
}
