"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getPodDetail } from "@/api/pod";
import { getCurrentClusterId } from "@/config/cluster";
import type { PodDetail, PodContainerDetail, PodVolume } from "@/types/cluster";
import {
  Box,
  Container,
  HardDrive,
  Network,
  Settings,
  Cpu,
  MemoryStick,
  Clock,
  Server,
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
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<PodDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!namespace || !podName) return;
    setLoading(true);
    setError("");
    try {
      const res = await getPodDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        PodName: podName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [namespace, podName]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: "概览", icon: <Box className="w-4 h-4" /> },
    { key: "containers", label: "容器", icon: <Container className="w-4 h-4" /> },
    { key: "volumes", label: "存储卷", icon: <HardDrive className="w-4 h-4" /> },
    { key: "network", label: "网络", icon: <Network className="w-4 h-4" /> },
    { key: "scheduling", label: "调度", icon: <Settings className="w-4 h-4" /> },
  ];

  return (
    <Modal
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
            {activeTab === "containers" && (
              <ContainersTab containers={detail.containers} onViewLogs={onViewLogs} />
            )}
            {activeTab === "volumes" && <VolumesTab volumes={detail.volumes || []} />}
            {activeTab === "network" && <NetworkTab detail={detail} />}
            {activeTab === "scheduling" && <SchedulingTab detail={detail} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

// 概览 Tab
function OverviewTab({ detail }: { detail: PodDetail }) {
  const infoItems = [
    { label: "命名空间", value: detail.namespace },
    { label: "状态", value: <StatusBadge status={detail.phase} /> },
    { label: "Ready", value: detail.ready },
    { label: "重启次数", value: detail.restarts },
    { label: "节点", value: detail.node },
    { label: "Pod IP", value: detail.podIP || "-" },
    { label: "QoS Class", value: detail.qosClass || "-" },
    { label: "Age", value: detail.age || "-" },
    { label: "Controller", value: detail.controller || "-" },
    { label: "Service Account", value: detail.serviceAccountName || "-" },
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

      {/* 资源使用 */}
      {(detail.cpuUsage || detail.memUsage) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">资源使用</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <Cpu className="w-4 h-4 text-orange-500" />
                <span className="text-sm font-medium text-default">CPU</span>
              </div>
              <div className="text-lg font-semibold text-default">
                {detail.cpuUsage || "0"} / {detail.cpuLimit || "unlimited"}
              </div>
              {detail.cpuUtilPct !== undefined && (
                <div className="mt-2">
                  <div className="h-2 bg-[var(--border-color)] rounded-full overflow-hidden">
                    <div
                      className="h-full bg-orange-500 transition-all duration-300"
                      style={{ width: `${Math.min(detail.cpuUtilPct, 100)}%` }}
                    />
                  </div>
                  <div className="text-xs text-muted mt-1">{detail.cpuUtilPct.toFixed(1)}%</div>
                </div>
              )}
            </div>

            <div className="bg-[var(--background)] rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <MemoryStick className="w-4 h-4 text-green-500" />
                <span className="text-sm font-medium text-default">Memory</span>
              </div>
              <div className="text-lg font-semibold text-default">
                {detail.memUsage || "0"} / {detail.memLimit || "unlimited"}
              </div>
              {detail.memUtilPct !== undefined && (
                <div className="mt-2">
                  <div className="h-2 bg-[var(--border-color)] rounded-full overflow-hidden">
                    <div
                      className="h-full bg-green-500 transition-all duration-300"
                      style={{ width: `${Math.min(detail.memUtilPct, 100)}%` }}
                    />
                  </div>
                  <div className="text-xs text-muted mt-1">{detail.memUtilPct.toFixed(1)}%</div>
                </div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* Badges */}
      {detail.badges && detail.badges.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">标签</h3>
          <div className="flex flex-wrap gap-2">
            {detail.badges.map((badge, i) => (
              <span
                key={i}
                className="px-2 py-1 bg-primary/10 text-primary text-xs rounded"
              >
                {badge}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* 状态消息 */}
      {(detail.reason || detail.message) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">状态信息</h3>
          <div className="bg-[var(--background)] rounded-lg p-4">
            {detail.reason && (
              <div className="text-sm text-yellow-600 dark:text-yellow-400 font-medium">
                {detail.reason}
              </div>
            )}
            {detail.message && <div className="text-sm text-muted mt-1">{detail.message}</div>}
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
}: {
  containers: PodContainerDetail[];
  onViewLogs: (name: string) => void;
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
              查看日志
            </button>
          </div>

          {/* 容器详情 */}
          <div className="p-4 space-y-4">
            {/* 镜像 */}
            <div>
              <div className="text-xs text-muted mb-1">镜像</div>
              <div className="text-sm text-default font-mono bg-[var(--background)] px-2 py-1 rounded break-all">
                {container.image}
              </div>
            </div>

            {/* 资源配置 */}
            <div className="grid grid-cols-2 gap-4">
              <div>
                <div className="text-xs text-muted mb-1">Requests</div>
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
                <div className="text-xs text-muted mb-1">Limits</div>
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
                <div className="text-xs text-muted mb-1">端口</div>
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
                  <span className="text-muted">重启次数: </span>
                  <span className="text-yellow-600 dark:text-yellow-400 font-medium">
                    {container.restartCount}
                  </span>
                </div>
                {container.lastTerminatedReason && (
                  <div>
                    <span className="text-muted">上次终止原因: </span>
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
function VolumesTab({ volumes }: { volumes: PodVolume[] }) {
  if (volumes.length === 0) {
    return <div className="text-center py-8 text-muted">暂无存储卷</div>;
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
function NetworkTab({ detail }: { detail: PodDetail }) {
  const networkItems = [
    { label: "Pod IP", value: detail.podIP || "-" },
    { label: "Host IP", value: detail.hostIP || "-" },
    { label: "Host Network", value: detail.hostNetwork ? "是" : "否" },
    { label: "DNS Policy", value: detail.dnsPolicy || "-" },
    { label: "Service Account", value: detail.serviceAccountName || "-" },
  ];

  return (
    <div className="space-y-6">
      <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
        {networkItems.map((item) => (
          <div key={item.label} className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{item.label}</div>
            <div className="text-sm text-default font-medium">{item.value}</div>
          </div>
        ))}
      </div>

      {/* Pod IPs */}
      {detail.podIPs && detail.podIPs.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Pod IPs</h3>
          <div className="flex flex-wrap gap-2">
            {detail.podIPs.map((ip, i) => (
              <span
                key={i}
                className="px-3 py-1.5 bg-[var(--background)] text-sm font-mono rounded"
              >
                {ip}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// 调度 Tab
function SchedulingTab({ detail }: { detail: PodDetail }) {
  const schedulingItems = [
    { label: "节点", value: detail.node },
    { label: "重启策略", value: detail.restartPolicy || "-" },
    { label: "优先级类", value: detail.priorityClassName || "-" },
    { label: "运行时类", value: detail.runtimeClassName || "-" },
    {
      label: "终止宽限期",
      value: detail.terminationGracePeriodSeconds
        ? `${detail.terminationGracePeriodSeconds}s`
        : "-",
    },
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

      {/* Node Selector */}
      {detail.nodeSelector && Object.keys(detail.nodeSelector).length > 0 ? (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Node Selector</h3>
          <div className="bg-[var(--background)] rounded-lg p-4">
            {Object.entries(detail.nodeSelector).map(([k, v]) => (
              <div key={k} className="text-sm">
                <span className="text-muted">{k}: </span>
                <span className="text-default">{v}</span>
              </div>
            ))}
          </div>
        </div>
      ) : null}

      {/* Tolerations */}
      {Array.isArray(detail.tolerations) && detail.tolerations.length > 0 ? (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Tolerations</h3>
          <div className="bg-[var(--background)] rounded-lg p-4 text-sm font-mono overflow-x-auto">
            <pre className="text-xs">{JSON.stringify(detail.tolerations, null, 2)}</pre>
          </div>
        </div>
      ) : null}
    </div>
  );
}
