"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getPodDetail } from "@/datasource/cluster";
import { useClusterStore } from "@/store/clusterStore";
import { useI18n } from "@/i18n/context";
import type { PodDetail, PodContainerDetail, PodVolume } from "@/types/cluster";
import {
  Box,
  Container,
  HardDrive,
  Network,
  Settings,
  Cpu,
  MemoryStick,
} from "lucide-react";

interface PodDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  podName: string;
  onViewLogs: (containerName: string) => void;
}

type TabType = "overview" | "containers" | "volumes" | "network" | "scheduling";

export function PodDetailModal({
  isOpen,
  onClose,
  namespace,
  podName,
  onViewLogs,
}: PodDetailModalProps) {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<PodDetail | null>(null);
  const { currentClusterId } = useClusterStore();

  const fetchDetail = useCallback(async () => {
    if (!namespace || !podName || !currentClusterId) return;
    setLoading(true);
    setError("");
    try {
      const res = await getPodDetail({
        ClusterID: currentClusterId,
        Namespace: namespace,
        PodName: podName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [namespace, podName, currentClusterId, t.common.loadFailed]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.pod.overview, icon: <Box className="w-4 h-4" /> },
    { key: "containers", label: t.pod.containers, icon: <Container className="w-4 h-4" /> },
    { key: "volumes", label: t.pod.volumeMounts, icon: <HardDrive className="w-4 h-4" /> },
    { key: "network", label: t.pod.network, icon: <Network className="w-4 h-4" /> },
    { key: "scheduling", label: t.pod.scheduling, icon: <Settings className="w-4 h-4" /> },
  ];

  return (
    <Drawer
      isOpen={isOpen}
      onClose={onClose}
      title={`Pod: ${podName}`}
      size="xl"
    >
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
            {activeTab === "containers" && (
              <ContainersTab containers={detail.containers} onViewLogs={onViewLogs} t={t} />
            )}
            {activeTab === "volumes" && <VolumesTab volumes={detail.volumes || []} t={t} />}
            {activeTab === "network" && <NetworkTab detail={detail} t={t} />}
            {activeTab === "scheduling" && <SchedulingTab detail={detail} t={t} />}
          </div>
        </div>
      ) : null}
    </Drawer>
  );
}

// 概览 Tab
function OverviewTab({ detail, t }: { detail: PodDetail; t: ReturnType<typeof useI18n>["t"] }) {
  // 计算运行时间
  const getAge = (startTime?: string) => {
    if (!startTime) return "-";
    const start = new Date(startTime);
    const now = new Date();
    const diffMs = now.getTime() - start.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    const diffHours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const diffMins = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));

    if (diffDays > 0) return `${diffDays}d${diffHours}h`;
    if (diffHours > 0) return `${diffHours}h${diffMins}m`;
    return `${diffMins}m`;
  };

  const infoItems = [
    { label: t.common.namespace, value: detail.namespace },
    { label: t.common.status, value: <StatusBadge status={detail.phase} /> },
    { label: t.pod.ready, value: detail.ready || "-" },
    { label: t.pod.restartCount, value: detail.restarts ?? 0 },
    { label: t.pod.node, value: detail.node || "-" },
    { label: t.pod.ip, value: detail.podIP || "-" },
    { label: t.pod.hostIP, value: detail.hostIP || "-" },
    { label: t.pod.age, value: getAge(detail.startTime) },
    { label: t.pod.controller, value: detail.controller || "-" },
  ];

  return (
    <div className="space-y-6">
      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.pod.basicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {infoItems.map((item) => (
            <div key={item.label} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 资源使用 */}
      {(detail.cpuUsage || detail.memUsage) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.pod.resourceUsage}</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <Cpu className="w-4 h-4 text-orange-500" />
                <span className="text-sm font-medium text-default">CPU</span>
              </div>
              <div className="text-2xl font-semibold text-default">
                {detail.cpuUsage || "-"}
              </div>
            </div>

            <div className="bg-[var(--background)] rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <MemoryStick className="w-4 h-4 text-green-500" />
                <span className="text-sm font-medium text-default">Memory</span>
              </div>
              <div className="text-2xl font-semibold text-default">
                {detail.memUsage || "-"}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 容器摘要 */}
      {detail.containers && detail.containers.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">
            {t.pod.containers} ({detail.containers.length})
          </h3>
          <div className="space-y-2">
            {detail.containers.map((c) => (
              <div
                key={c.name}
                className="bg-[var(--background)] rounded-lg p-3 flex items-center justify-between"
              >
                <div className="flex items-center gap-3">
                  <Container className="w-4 h-4 text-muted" />
                  <span className="text-sm font-medium text-default">{c.name}</span>
                </div>
                <div className="flex items-center gap-2">
                  {c.restartCount !== undefined && c.restartCount > 0 && (
                    <span className="text-xs text-yellow-600 dark:text-yellow-400">
                      {t.pod.restartTimes.replace("{count}", String(c.restartCount))}
                    </span>
                  )}
                  <StatusBadge status={c.state === "running" ? "Running" : c.state || "Unknown"} />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// 容器 Tab
function ContainersTab({
  containers,
  onViewLogs,
  t,
}: {
  containers: PodContainerDetail[];
  onViewLogs: (name: string) => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  return (
    <div className="space-y-4">
      {containers.map((container) => (
        <div
          key={container.name}
          className="border border-[var(--border-color)] rounded-lg overflow-hidden"
        >
          {/* 容器头部 */}
          <div className="bg-[var(--background)] px-4 py-3 flex items-center justify-between">
            <div className="flex items-center gap-3">
              <Container className="w-5 h-5 text-muted" />
              <span className="font-medium text-default">{container.name}</span>
              {container.state && (
                <StatusBadge status={container.state === "running" ? "Running" : container.state} />
              )}
            </div>
            <button
              onClick={() => onViewLogs(container.name)}
              className="px-3 py-1.5 text-xs font-medium text-primary hover:bg-primary/10 rounded transition-colors"
            >
              {t.pod.viewLogs}
            </button>
          </div>

          {/* 容器详情 */}
          <div className="p-4 space-y-4">
            {/* 镜像 */}
            <div>
              <div className="text-xs text-muted mb-1">{t.pod.image}</div>
              <div className="text-sm text-default font-mono bg-[var(--background)] px-2 py-1 rounded break-all">
                {container.image}
              </div>
            </div>

            {/* 资源配置 */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-xs text-muted mb-1">{t.pod.requests}</div>
                <div className="text-sm text-default">
                  {container.requests ? (
                    <div className="space-y-1">
                      {Object.entries(container.requests).map(([k, v]) => (
                        <div key={k}>
                          {k}: {v}
                        </div>
                      ))}
                    </div>
                  ) : (
                    "-"
                  )}
                </div>
              </div>
              <div>
                <div className="text-xs text-muted mb-1">{t.pod.limits}</div>
                <div className="text-sm text-default">
                  {container.limits ? (
                    <div className="space-y-1">
                      {Object.entries(container.limits).map(([k, v]) => (
                        <div key={k}>
                          {k}: {v}
                        </div>
                      ))}
                    </div>
                  ) : (
                    "-"
                  )}
                </div>
              </div>
            </div>

            {/* 端口 */}
            {container.ports && container.ports.length > 0 && (
              <div>
                <div className="text-xs text-muted mb-1">{t.pod.ports}</div>
                <div className="flex flex-wrap gap-2">
                  {container.ports.map((port, i) => (
                    <span
                      key={i}
                      className="px-2 py-1 bg-[var(--background)] text-xs rounded font-mono"
                    >
                      {port.containerPort}/{port.protocol}
                      {port.name && ` (${port.name})`}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* 重启信息 */}
            {container.restartCount !== undefined && container.restartCount > 0 && (
              <div className="flex items-center gap-4 text-sm">
                <div>
                  <span className="text-muted">{t.pod.restartCount}: </span>
                  <span className="text-yellow-600 dark:text-yellow-400 font-medium">
                    {container.restartCount}
                  </span>
                </div>
                {container.lastTerminatedReason && (
                  <div>
                    <span className="text-muted">{t.pod.lastTerminatedReason}: </span>
                    <span className="text-default">{container.lastTerminatedReason}</span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

// 存储卷 Tab
function VolumesTab({ volumes, t }: { volumes: PodVolume[]; t: ReturnType<typeof useI18n>["t"] }) {
  if (volumes.length === 0) {
    return <div className="text-center py-8 text-muted">{t.pod.noVolumes}</div>;
  }

  return (
    <div className="space-y-3">
      {volumes.map((volume) => (
        <div
          key={volume.name}
          className="bg-[var(--background)] rounded-lg p-4 flex items-center justify-between"
        >
          <div className="flex items-center gap-3">
            <HardDrive className="w-5 h-5 text-muted" />
            <div>
              <div className="font-medium text-default">{volume.name}</div>
              <div className="text-xs text-muted">{volume.sourceBrief || volume.type}</div>
            </div>
          </div>
          <span className="px-2 py-1 bg-card text-xs text-muted rounded">{volume.type}</span>
        </div>
      ))}
    </div>
  );
}

// 网络 Tab
function NetworkTab({ detail, t }: { detail: PodDetail; t: ReturnType<typeof useI18n>["t"] }) {
  const networkItems = [
    { label: t.pod.ip, value: detail.podIP || "-" },
    { label: t.pod.hostIP, value: detail.hostIP || "-" },
    { label: t.pod.node, value: detail.node || "-" },
  ];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        {networkItems.map((item) => (
          <div key={item.label} className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{item.label}</div>
            <div className="text-sm text-default font-medium font-mono">{item.value}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

// 调度 Tab
function SchedulingTab({ detail, t }: { detail: PodDetail; t: ReturnType<typeof useI18n>["t"] }) {
  // 计算运行时间
  const getAge = (startTime?: string) => {
    if (!startTime) return "-";
    const start = new Date(startTime);
    const now = new Date();
    const diffMs = now.getTime() - start.getTime();
    const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
    const diffHours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const diffMins = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60));

    if (diffDays > 0) return `${diffDays}d${diffHours}h`;
    if (diffHours > 0) return `${diffHours}h${diffMins}m`;
    return `${diffMins}m`;
  };

  const schedulingItems = [
    { label: t.pod.node, value: detail.node || "-" },
    { label: t.pod.controller, value: detail.controller || "-" },
    { label: t.common.createdAt, value: detail.startTime ? new Date(detail.startTime).toLocaleString() : "-" },
    { label: t.pod.age, value: getAge(detail.startTime) },
  ];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        {schedulingItems.map((item) => (
          <div key={item.label} className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{item.label}</div>
            <div className="text-sm text-default font-medium">{item.value}</div>
          </div>
        ))}
      </div>
    </div>
  );
}
