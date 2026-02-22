"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge, ConfirmDialog } from "@/components/common";
import { getNodeDetail, cordonNode, uncordonNode } from "@/datasource/cluster";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { useRequireAuth } from "@/hooks/useRequireAuth";
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
  Shield,
  ShieldOff,
} from "lucide-react";

interface NodeDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
  onNodeChanged?: () => void;
}

type TabType = "overview" | "resources" | "conditions" | "taints" | "labels";

export function NodeDetailModal({ isOpen, onClose, nodeName, onNodeChanged }: NodeDetailModalProps) {
  const { t } = useI18n();
  const requireAuth = useRequireAuth();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<NodeDetail | null>(null);

  // 封锁/解封状态
  const [showCordonConfirm, setShowCordonConfirm] = useState(false);
  const [cordonLoading, setCordonLoading] = useState(false);

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
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [nodeName, t.common.loadFailed]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  // 封锁/解封操作
  const handleCordonClick = () => {
    requireAuth(() => setShowCordonConfirm(true));
  };

  const handleCordonConfirm = async () => {
    if (!detail) return;
    setCordonLoading(true);
    try {
      if (detail.schedulable) {
        await cordonNode({ ClusterID: getCurrentClusterId(), Node: nodeName });
      } else {
        await uncordonNode({ ClusterID: getCurrentClusterId(), Node: nodeName });
      }
      setShowCordonConfirm(false);
      // 刷新详情 + 通知父组件
      setTimeout(() => {
        fetchDetail();
        onNodeChanged?.();
      }, 2000);
    } catch (err) {
      console.error("Cordon/Uncordon failed:", err);
    } finally {
      setCordonLoading(false);
    }
  };

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.node.overview, icon: <Server className="w-4 h-4" /> },
    { key: "resources", label: t.node.resources, icon: <Cpu className="w-4 h-4" /> },
    { key: "conditions", label: t.node.conditions, icon: <CheckCircle className="w-4 h-4" /> },
    { key: "taints", label: t.node.taints, icon: <AlertTriangle className="w-4 h-4" /> },
    { key: "labels", label: t.node.labels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <>
      <Drawer isOpen={isOpen} onClose={onClose} title={`Node: ${nodeName}`} size="xl">
        {loading ? (
          <div className="py-12">
            <LoadingSpinner />
          </div>
        ) : error ? (
          <div className="p-6 text-center text-red-500">{error}</div>
        ) : detail ? (
          <div className="flex flex-col h-full">
            {/* 操作栏 */}
            <div className="flex items-center gap-2 px-4 pt-2 pb-1 shrink-0">
              <button
                onClick={handleCordonClick}
                className={`flex items-center gap-2 px-3 py-1.5 text-sm rounded-lg border transition-colors ${
                  detail.schedulable
                    ? "border-yellow-300 dark:border-yellow-700 text-yellow-600 dark:text-yellow-400 hover:bg-yellow-50 dark:hover:bg-yellow-900/20"
                    : "border-green-300 dark:border-green-700 text-green-600 dark:text-green-400 hover:bg-green-50 dark:hover:bg-green-900/20"
                }`}
              >
                {detail.schedulable ? (
                  <Shield className="w-4 h-4" />
                ) : (
                  <ShieldOff className="w-4 h-4" />
                )}
                {detail.schedulable ? t.node.cordon : t.node.uncordon}
              </button>
            </div>

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
              {activeTab === "resources" && <ResourcesTab detail={detail} t={t} />}
              {activeTab === "conditions" && <ConditionsTab conditions={detail.conditions || []} t={t} />}
              {activeTab === "taints" && <TaintsTab taints={detail.taints || []} t={t} />}
              {activeTab === "labels" && <LabelsTab labels={detail.labels || {}} t={t} />}
            </div>
          </div>
        ) : null}
      </Drawer>

      {/* 封锁/解封确认对话框 */}
      <ConfirmDialog
        isOpen={showCordonConfirm}
        onClose={() => setShowCordonConfirm(false)}
        onConfirm={handleCordonConfirm}
        title={detail?.schedulable ? t.node.cordonConfirmTitle : t.node.uncordonConfirmTitle}
        message={
          detail?.schedulable
            ? t.node.cordonConfirmMessage.replace("{name}", nodeName)
            : t.node.uncordonConfirmMessage.replace("{name}", nodeName)
        }
        confirmText={detail?.schedulable ? t.node.cordon : t.node.uncordon}
        cancelText={t.common.cancel}
        loading={cordonLoading}
        variant={detail?.schedulable ? "warning" : "info"}
      />
    </>
  );
}

// 概览 Tab
function OverviewTab({ detail, t }: { detail: NodeDetail; t: ReturnType<typeof useI18n>["t"] }) {
  const infoItems = [
    { label: t.common.status, value: <StatusBadge status={detail.ready ? "Ready" : "NotReady"} /> },
    { label: t.node.schedulable, value: detail.schedulable ? t.common.yes : `${t.common.no}（${t.node.cordoned}）` },
    { label: t.node.role, value: detail.roles?.join(", ") || "-" },
    { label: "Age", value: detail.age || "-" },
    { label: t.node.hostname, value: detail.hostname || "-" },
    { label: t.node.internalIP, value: detail.internalIP || "-" },
    { label: t.node.externalIP, value: detail.externalIP || "-" },
    { label: t.node.osImage, value: detail.osImage || "-" },
    { label: t.node.architecture, value: detail.architecture || "-" },
    { label: t.node.kernelVersion, value: detail.kernel || "-" },
    { label: t.node.containerRuntime, value: detail.cri || "-" },
    { label: t.node.kubeletVersion, value: detail.kubelet || "-" },
  ];

  return (
    <div className="space-y-6">
      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.node.basicInfo}</h3>
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
          <h3 className="text-sm font-semibold text-default mb-3">{t.node.pressureWarning}</h3>
          <div className="flex flex-wrap gap-2">
            {detail.pressureMemory && (
              <span className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded">
                {t.node.memoryPressure}
              </span>
            )}
            {detail.pressureDisk && (
              <span className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded">
                {t.node.diskPressure}
              </span>
            )}
            {detail.pressurePID && (
              <span className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded">
                {t.node.pidPressure}
              </span>
            )}
            {detail.networkUnavailable && (
              <span className="px-3 py-1.5 bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded">
                {t.node.networkUnavailable}
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
function ResourcesTab({ detail, t }: { detail: NodeDetail; t: ReturnType<typeof useI18n>["t"] }) {
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
        <h3 className="text-sm font-semibold text-default mb-3">{t.node.resourceUsage}</h3>
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
        <h3 className="text-sm font-semibold text-default mb-3">{t.node.capacityAllocatable}</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.node.cpuCapacity}</div>
            <div className="text-sm text-default font-medium">{detail.cpuCapacityCores || 0} cores</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.node.cpuAllocatable}</div>
            <div className="text-sm text-default font-medium">{detail.cpuAllocatableCores || 0} cores</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.node.memoryCapacity}</div>
            <div className="text-sm text-default font-medium">{(detail.memCapacityGiB || 0).toFixed(1)} GiB</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.node.memoryAllocatable}</div>
            <div className="text-sm text-default font-medium">{(detail.memAllocatableGiB || 0).toFixed(1)} GiB</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.node.podCapacity}</div>
            <div className="text-sm text-default font-medium">{detail.podsCapacity || 0}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.node.podAllocatable}</div>
            <div className="text-sm text-default font-medium">{detail.podsAllocatable || 0}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.node.ephemeralStorage}</div>
            <div className="text-sm text-default font-medium">{(detail.ephemeralStorageGiB || 0).toFixed(1)} GiB</div>
          </div>
        </div>
      </div>
    </div>
  );
}

// 状态 Tab
function ConditionsTab({ conditions, t }: { conditions: NodeCondition[]; t: ReturnType<typeof useI18n>["t"] }) {
  if (conditions.length === 0) {
    return <div className="text-center py-8 text-muted">{t.node.noConditions}</div>;
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
function TaintsTab({ taints, t }: { taints: NodeTaint[]; t: ReturnType<typeof useI18n>["t"] }) {
  if (taints.length === 0) {
    return <div className="text-center py-8 text-muted">{t.node.noTaints}</div>;
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
function LabelsTab({ labels, t }: { labels: Record<string, string>; t: ReturnType<typeof useI18n>["t"] }) {
  const entries = Object.entries(labels);
  if (entries.length === 0) {
    return <div className="text-center py-8 text-muted">{t.node.noLabels}</div>;
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
