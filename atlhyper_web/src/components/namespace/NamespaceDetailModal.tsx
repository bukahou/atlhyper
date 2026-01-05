"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getNamespaceDetail, getConfigMaps } from "@/api/namespace";
import { getCurrentClusterId } from "@/config/cluster";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import type { NamespaceDetail, ConfigMapDTO, ResourceQuotaDTO, LimitRangeDTO } from "@/types/cluster";
import {
  FolderTree,
  Box,
  FileText,
  Tag,
  Shield,
  ChevronDown,
  ChevronRight,
} from "lucide-react";

interface NamespaceDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespaceName: string;
}

type TabType = "overview" | "quotas" | "configmaps" | "labels";

export function NamespaceDetailModal({
  isOpen,
  onClose,
  namespaceName,
}: NamespaceDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<NamespaceDetail | null>(null);
  const [configMaps, setConfigMaps] = useState<ConfigMapDTO[]>([]);
  const [configMapsLoading, setConfigMapsLoading] = useState(false);
  const requireAuth = useRequireAuth();

  const fetchDetail = useCallback(async () => {
    if (!namespaceName) return;
    setLoading(true);
    setError("");
    try {
      const res = await getNamespaceDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespaceName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [namespaceName]);

  const fetchConfigMaps = useCallback(async () => {
    if (!namespaceName) return;
    setConfigMapsLoading(true);
    try {
      const res = await getConfigMaps({
        ClusterID: getCurrentClusterId(),
        Namespace: namespaceName,
      });
      setConfigMaps(res.data.data || []);
    } catch (err) {
      console.error("Failed to fetch configmaps:", err);
      setConfigMaps([]);
    } finally {
      setConfigMapsLoading(false);
    }
  }, [namespaceName]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
      setConfigMaps([]);
    }
  }, [isOpen, fetchDetail]);

  // 当切换到 ConfigMap tab 时加载数据
  useEffect(() => {
    if (activeTab === "configmaps" && configMaps.length === 0 && !configMapsLoading) {
      fetchConfigMaps();
    }
  }, [activeTab, configMaps.length, configMapsLoading, fetchConfigMaps]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: "概览", icon: <FolderTree className="w-4 h-4" /> },
    { key: "quotas", label: "配额", icon: <Shield className="w-4 h-4" /> },
    { key: "configmaps", label: "ConfigMap", icon: <FileText className="w-4 h-4" /> },
    { key: "labels", label: "标签", icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Namespace: ${namespaceName}`} size="xl">
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
            {activeTab === "quotas" && (
              <QuotasTab quotas={detail.quotas || []} limitRanges={detail.limitRanges || []} />
            )}
            {activeTab === "configmaps" && (
              <ConfigMapsTab configMaps={configMaps} loading={configMapsLoading} requireAuth={requireAuth} />
            )}
            {activeTab === "labels" && (
              <LabelsTab labels={detail.labels || {}} annotations={detail.annotations || {}} />
            )}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

// 概览 Tab
function OverviewTab({ detail }: { detail: NamespaceDetail }) {
  const basicInfo = [
    { label: "名称", value: detail.name },
    { label: "状态", value: <StatusBadge status={detail.phase} /> },
    { label: "Age", value: detail.age || "-" },
    { label: "创建时间", value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
    { label: "Labels", value: detail.labelCount },
    { label: "Annotations", value: detail.annotationCount },
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
        <h3 className="text-sm font-semibold text-default mb-3">基本信息</h3>
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
        <h3 className="text-sm font-semibold text-default mb-3">Pod 状态</h3>
        <div className="grid grid-cols-4 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-default">{detail.pods}</div>
            <div className="text-xs text-muted mt-1">总计</div>
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
        <h3 className="text-sm font-semibold text-default mb-3">工作负载</h3>
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
          <h3 className="text-sm font-semibold text-default mb-3">网络</h3>
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
          <h3 className="text-sm font-semibold text-default mb-3">配置</h3>
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
          <h3 className="text-sm font-semibold text-default mb-3">资源使用</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-4">
              <div className="text-sm font-medium mb-2">CPU</div>
              <div className="text-lg font-bold text-default">{detail.metrics.cpu.usage}</div>
              {detail.metrics.cpu.utilPct !== undefined && (
                <div className="text-xs text-muted mt-1">{detail.metrics.cpu.utilPct.toFixed(1)}% 使用率</div>
              )}
            </div>
            <div className="bg-[var(--background)] rounded-lg p-4">
              <div className="text-sm font-medium mb-2">Memory</div>
              <div className="text-lg font-bold text-default">{detail.metrics.memory.usage}</div>
              {detail.metrics.memory.utilPct !== undefined && (
                <div className="text-xs text-muted mt-1">{detail.metrics.memory.utilPct.toFixed(1)}% 使用率</div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// 配额 Tab
function QuotasTab({ quotas, limitRanges }: { quotas: ResourceQuotaDTO[]; limitRanges: LimitRangeDTO[] }) {
  if (quotas.length === 0 && limitRanges.length === 0) {
    return <div className="text-center py-8 text-muted">暂无配额或限制范围</div>;
  }

  return (
    <div className="space-y-6">
      {/* Resource Quotas */}
      {quotas.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Resource Quotas ({quotas.length})</h3>
          <div className="space-y-3">
            {quotas.map((quota, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-4">
                <h4 className="font-medium text-default mb-3">{quota.name}</h4>
                {quota.scopes && quota.scopes.length > 0 && (
                  <div className="mb-3">
                    <span className="text-xs text-muted">Scopes: </span>
                    {quota.scopes.map((scope, j) => (
                      <StatusBadge key={j} status={scope} type="info" />
                    ))}
                  </div>
                )}
                {quota.hard && Object.keys(quota.hard).length > 0 && (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                    {Object.entries(quota.hard).map(([key, value]) => (
                      <div key={key} className="bg-card rounded p-2">
                        <div className="text-xs text-muted">{key}</div>
                        <div className="text-sm font-mono">
                          <span className="text-default">{quota.used?.[key] || "0"}</span>
                          <span className="text-muted"> / {value}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Limit Ranges */}
      {limitRanges.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Limit Ranges ({limitRanges.length})</h3>
          <div className="space-y-3">
            {limitRanges.map((lr, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-4">
                <h4 className="font-medium text-default mb-3">{lr.name}</h4>
                <div className="space-y-2">
                  {lr.items.map((item, j) => (
                    <div key={j} className="bg-card rounded p-3">
                      <StatusBadge status={item.type} type="info" />
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-2 mt-2 text-xs">
                        {item.default && Object.keys(item.default).length > 0 && (
                          <div>
                            <span className="text-muted">Default: </span>
                            {Object.entries(item.default).map(([k, v]) => (
                              <span key={k} className="font-mono">{k}={v} </span>
                            ))}
                          </div>
                        )}
                        {item.defaultRequest && Object.keys(item.defaultRequest).length > 0 && (
                          <div>
                            <span className="text-muted">Request: </span>
                            {Object.entries(item.defaultRequest).map(([k, v]) => (
                              <span key={k} className="font-mono">{k}={v} </span>
                            ))}
                          </div>
                        )}
                        {item.max && Object.keys(item.max).length > 0 && (
                          <div>
                            <span className="text-muted">Max: </span>
                            {Object.entries(item.max).map(([k, v]) => (
                              <span key={k} className="font-mono">{k}={v} </span>
                            ))}
                          </div>
                        )}
                        {item.min && Object.keys(item.min).length > 0 && (
                          <div>
                            <span className="text-muted">Min: </span>
                            {Object.entries(item.min).map(([k, v]) => (
                              <span key={k} className="font-mono">{k}={v} </span>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// ConfigMap Tab
function ConfigMapsTab({
  configMaps,
  loading,
  requireAuth,
}: {
  configMaps: ConfigMapDTO[];
  loading: boolean;
  requireAuth: (action: () => void) => boolean;
}) {
  const [expandedCMs, setExpandedCMs] = useState<Set<string>>(new Set());

  const toggleExpand = (name: string) => {
    const isExpanded = expandedCMs.has(name);
    if (isExpanded) {
      // 收起不需要检查登录
      setExpandedCMs((prev) => {
        const next = new Set(prev);
        next.delete(name);
        return next;
      });
    } else {
      // 展开需要检查登录
      requireAuth(() => {
        setExpandedCMs((prev) => {
          const next = new Set(prev);
          next.add(name);
          return next;
        });
      });
    }
  };

  if (loading) {
    return (
      <div className="py-8">
        <LoadingSpinner />
      </div>
    );
  }

  if (configMaps.length === 0) {
    return <div className="text-center py-8 text-muted">暂无 ConfigMap</div>;
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
  };

  return (
    <div className="space-y-3">
      {configMaps.map((cm) => {
        const isExpanded = expandedCMs.has(cm.name);
        return (
          <div key={cm.name} className="bg-[var(--background)] rounded-lg overflow-hidden">
            <button
              onClick={() => toggleExpand(cm.name)}
              className="w-full p-4 flex items-center justify-between hover:bg-[var(--border-color)]/30 transition-colors"
            >
              <div className="flex items-center gap-3">
                <FileText className="w-5 h-5 text-primary" />
                <div className="text-left">
                  <div className="font-medium text-default">{cm.name}</div>
                  <div className="text-xs text-muted">
                    {cm.keys} keys · {formatBytes(cm.totalSizeBytes)}
                    {cm.immutable && <span className="ml-2 text-yellow-500">(Immutable)</span>}
                  </div>
                </div>
              </div>
              {isExpanded ? (
                <ChevronDown className="w-4 h-4 text-muted" />
              ) : (
                <ChevronRight className="w-4 h-4 text-muted" />
              )}
            </button>

            {isExpanded && cm.data && cm.data.length > 0 && (
              <div className="px-4 pb-4 border-t border-[var(--border-color)]">
                <div className="mt-3 space-y-2">
                  {cm.data.map((entry, i) => (
                    <div key={i} className="bg-card rounded p-3">
                      <div className="flex items-center justify-between mb-1">
                        <span className="font-mono text-sm text-primary">{entry.key}</span>
                        <span className="text-xs text-muted">{entry.size} bytes</span>
                      </div>
                      {entry.preview && (
                        <pre className="text-xs text-muted bg-[var(--background)] p-2 rounded overflow-x-auto max-h-24">
                          {entry.preview}
                          {entry.truncated && <span className="text-yellow-500">...</span>}
                        </pre>
                      )}
                    </div>
                  ))}
                </div>
                {cm.binary && cm.binary.length > 0 && (
                  <div className="mt-3">
                    <div className="text-xs text-muted mb-2">Binary Data ({cm.binaryKeys} keys)</div>
                    <div className="flex flex-wrap gap-2">
                      {cm.binary.map((entry, i) => (
                        <span key={i} className="px-2 py-1 bg-card rounded text-xs font-mono">
                          {entry.key} ({formatBytes(entry.size)})
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

// 标签 Tab
function LabelsTab({ labels, annotations }: { labels: Record<string, string>; annotations: Record<string, string> }) {
  const labelEntries = Object.entries(labels);
  const annoEntries = Object.entries(annotations);

  return (
    <div className="space-y-6">
      {/* Labels */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Labels ({labelEntries.length})</h3>
        {labelEntries.length === 0 ? (
          <div className="text-center py-4 text-muted">暂无标签</div>
        ) : (
          <div className="space-y-2">
            {labelEntries.map(([key, value]) => (
              <div key={key} className="bg-[var(--background)] rounded-lg p-3 flex items-start gap-2">
                <span className="text-sm font-mono text-primary break-all">{key}</span>
                <span className="text-muted">=</span>
                <span className="text-sm font-mono text-default break-all">{value || '""'}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Annotations */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Annotations ({annoEntries.length})</h3>
        {annoEntries.length === 0 ? (
          <div className="text-center py-4 text-muted">暂无注解</div>
        ) : (
          <div className="space-y-2">
            {annoEntries.map(([key, value]) => (
              <div key={key} className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-sm font-mono text-primary break-all mb-1">{key}</div>
                <div className="text-sm font-mono text-default break-all whitespace-pre-wrap">{value || '""'}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
