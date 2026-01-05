"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getNodeDetail } from "@/api/node";
import { getCurrentClusterId } from "@/config/cluster";
import type { NodeDetail, NodeCondition, NodeTaint } from "@/types/cluster";
import {
  Server,
  Cpu,
  MemoryStick,
  HardDrive,
  Network,
  Tag,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Box,
} from "lucide-react";

interface NodeDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
}

type TabType = "overview" | "resources" | "conditions" | "taints" | "labels";

export function NodeDetailModal({ isOpen, onClose, nodeName }: NodeDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<NodeDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!nodeName) return;
    setLoading(true);
    setError("");
    try {
      const res = await getNodeDetail({
        ClusterID: getCurrentClusterId(),
        NodeName: nodeName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [nodeName]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: "概览", icon: <Server className="w-4 h-4" /> },
    { key: "resources", label: "资源", icon: <Cpu className="w-4 h-4" /> },
    { key: "conditions", label: "状态", icon: <CheckCircle className="w-4 h-4" /> },
    { key: "taints", label: "污点", icon: <AlertTriangle className="w-4 h-4" /> },
    { key: "labels", label: "标签", icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Node: ${nodeName}`} size="xl">
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
            {activeTab === "resources" && <ResourcesTab detail={detail} />}
            {activeTab === "conditions" && <ConditionsTab conditions={detail.conditions || []} />}
            {activeTab === "taints" && <TaintsTab taints={detail.taints || []} />}
            {activeTab === "labels" && <LabelsTab labels={detail.labels || {}} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

// 概览 Tab
function OverviewTab({ detail }: { detail: NodeDetail }) {
  const infoItems = [
    { label: "状态", value: <StatusBadge status={detail.ready ? "Ready" : "NotReady"} /> },
    { label: "可调度", value: detail.schedulable ? "是" : "否（已封锁）" },
    { label: "角色", value: detail.roles?.join(", ") || "-" },
    { label: "Age", value: detail.age || "-" },
    { label: "Hostname", value: detail.hostname || "-" },
    { label: "Internal IP", value: detail.internalIP || "-" },
    { label: "External IP", value: detail.externalIP || "-" },
    { label: "OS", value: detail.osImage || "-" },
    { label: "架构", value: detail.architecture || "-" },
    { label: "内核", value: detail.kernel || "-" },
    { label: "容器运行时", value: detail.cri || "-" },
    { label: "Kubelet", value: detail.kubelet || "-" },
  ];

  return (
    <div className="space-y-6">
      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">基本信息</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {infoItems.map((item) => (
            <div key={item.label} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 压力状态 */}
      {(detail.pressureMemory || detail.pressureDisk || detail.pressurePID || detail.networkUnavailable) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">压力告警</h3>
          <div className="flex flex-wrap gap-2">
            {detail.pressureMemory && (
              <span className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded">
                内存压力
              </span>
            )}
            {detail.pressureDisk && (
              <span className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded">
                磁盘压力
              </span>
            )}
            {detail.pressurePID && (
              <span className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded">
                PID 压力
              </span>
            )}
            {detail.networkUnavailable && (
              <span className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded">
                网络不可用
              </span>
            )}
          </div>
        </div>
      )}

      {/* Pod CIDR */}
      {detail.podCIDRs && detail.podCIDRs.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Pod CIDR</h3>
          <div className="flex flex-wrap gap-2">
            {detail.podCIDRs.map((cidr, i) => (
              <span key={i} className="px-3 py-1.5 bg-[var(--background)] text-sm font-mono rounded">
                {cidr}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// 资源 Tab
function ResourcesTab({ detail }: { detail: NodeDetail }) {
  const ResourceBar = ({
    label,
    icon,
    used,
    total,
    unit,
    color,
  }: {
    label: string;
    icon: React.ReactNode;
    used: number;
    total: number;
    unit: string;
    color: string;
  }) => {
    const percent = total > 0 ? (used / total) * 100 : 0;
    return (
      <div className="bg-[var(--background)] rounded-lg p-4">
        <div className="flex items-center gap-2 mb-3">
          {icon}
          <span className="text-sm font-medium text-default">{label}</span>
        </div>
        <div className="text-lg font-semibold text-default mb-2">
          {used.toFixed(1)} / {total.toFixed(1)} {unit}
        </div>
        <div className="h-2 bg-[var(--border-color)] rounded-full overflow-hidden">
          <div
            className={`h-full transition-all duration-300 ${color}`}
            style={{ width: `${Math.min(percent, 100)}%` }}
          />
        </div>
        <div className="text-xs text-muted mt-1">{percent.toFixed(1)}%</div>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      {/* 使用情况 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">资源使用</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <ResourceBar
            label="CPU"
            icon={<Cpu className="w-4 h-4 text-orange-500" />}
            used={detail.cpuUsageCores || 0}
            total={detail.cpuAllocatableCores || detail.cpuCapacityCores || 0}
            unit="cores"
            color="bg-orange-500"
          />
          <ResourceBar
            label="Memory"
            icon={<MemoryStick className="w-4 h-4 text-green-500" />}
            used={detail.memUsageGiB || 0}
            total={detail.memAllocatableGiB || detail.memCapacityGiB || 0}
            unit="GiB"
            color="bg-green-500"
          />
          <ResourceBar
            label="Pods"
            icon={<Box className="w-4 h-4 text-blue-500" />}
            used={detail.podsUsed || 0}
            total={detail.podsAllocatable || detail.podsCapacity || 0}
            unit=""
            color="bg-blue-500"
          />
        </div>
      </div>

      {/* 容量信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">容量 / 可分配</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">CPU 容量</div>
            <div className="text-sm text-default font-medium">{detail.cpuCapacityCores || 0} cores</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">CPU 可分配</div>
            <div className="text-sm text-default font-medium">{detail.cpuAllocatableCores || 0} cores</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">内存容量</div>
            <div className="text-sm text-default font-medium">{(detail.memCapacityGiB || 0).toFixed(1)} GiB</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">内存可分配</div>
            <div className="text-sm text-default font-medium">{(detail.memAllocatableGiB || 0).toFixed(1)} GiB</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Pod 容量</div>
            <div className="text-sm text-default font-medium">{detail.podsCapacity || 0}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Pod 可分配</div>
            <div className="text-sm text-default font-medium">{detail.podsAllocatable || 0}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">临时存储</div>
            <div className="text-sm text-default font-medium">{(detail.ephemeralStorageGiB || 0).toFixed(1)} GiB</div>
          </div>
        </div>
      </div>
    </div>
  );
}

// 状态 Tab
function ConditionsTab({ conditions }: { conditions: NodeCondition[] }) {
  if (conditions.length === 0) {
    return <div className="text-center py-8 text-muted">暂无状态信息</div>;
  }

  const getStatusIcon = (status: string) => {
    if (status === "True") return <CheckCircle className="w-4 h-4 text-green-500" />;
    if (status === "False") return <XCircle className="w-4 h-4 text-gray-400" />;
    return <AlertTriangle className="w-4 h-4 text-yellow-500" />;
  };

  return (
    <div className="space-y-3">
      {conditions.map((cond, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center gap-3 mb-2">
            {getStatusIcon(cond.status)}
            <span className="font-medium text-default">{cond.type}</span>
            <span
              className={`px-2 py-0.5 text-xs rounded ${
                cond.status === "True"
                  ? "bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400"
                  : cond.status === "False"
                  ? "bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400"
                  : "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400"
              }`}
            >
              {cond.status}
            </span>
          </div>
          {cond.reason && <div className="text-sm text-muted">Reason: {cond.reason}</div>}
          {cond.message && <div className="text-sm text-muted mt-1">{cond.message}</div>}
        </div>
      ))}
    </div>
  );
}

// 污点 Tab
function TaintsTab({ taints }: { taints: NodeTaint[] }) {
  if (taints.length === 0) {
    return <div className="text-center py-8 text-muted">暂无污点</div>;
  }

  const effectColors: Record<string, string> = {
    NoSchedule: "bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400",
    PreferNoSchedule: "bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400",
    NoExecute: "bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400",
  };

  return (
    <div className="space-y-3">
      {taints.map((taint, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4 flex items-center justify-between">
          <div>
            <div className="font-medium text-default font-mono">{taint.key}</div>
            {taint.value && <div className="text-sm text-muted">= {taint.value}</div>}
          </div>
          <span className={`px-2 py-1 text-xs rounded ${effectColors[taint.effect] || "bg-gray-100 text-gray-600"}`}>
            {taint.effect}
          </span>
        </div>
      ))}
    </div>
  );
}

// 标签 Tab
function LabelsTab({ labels }: { labels: Record<string, string> }) {
  const entries = Object.entries(labels);
  if (entries.length === 0) {
    return <div className="text-center py-8 text-muted">暂无标签</div>;
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
