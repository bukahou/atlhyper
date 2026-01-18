"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getStatefulSetDetail, type StatefulSetDetail } from "@/api/workload";
import { getCurrentClusterId } from "@/config/cluster";
import {
  Server,
  Box,
  Settings,
  Database,
  Tag,
  Activity,
  CheckCircle,
  AlertTriangle,
} from "lucide-react";

interface StatefulSetDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

type TabType = "overview" | "containers" | "strategy" | "storage" | "labels";

export function StatefulSetDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: StatefulSetDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<StatefulSetDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!name || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getStatefulSetDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Name: name,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
    } finally {
      setLoading(false);
    }
  }, [namespace, name]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: "概览", icon: <Server className="w-4 h-4" /> },
    { key: "containers", label: "容器", icon: <Box className="w-4 h-4" /> },
    { key: "strategy", label: "策略", icon: <Settings className="w-4 h-4" /> },
    { key: "storage", label: "存储", icon: <Database className="w-4 h-4" /> },
    { key: "labels", label: "标签", icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`StatefulSet: ${name}`} size="xl">
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
            {activeTab === "containers" && <ContainersTab detail={detail} />}
            {activeTab === "strategy" && <StrategyTab detail={detail} />}
            {activeTab === "storage" && <StorageTab detail={detail} />}
            {activeTab === "labels" && <LabelsTab detail={detail} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

// 概览 Tab
function OverviewTab({ detail }: { detail: StatefulSetDetail }) {
  const getRolloutBadgeType = (phase: string): "success" | "warning" | "error" | "info" => {
    switch (phase) {
      case "Complete": return "success";
      case "Progressing": return "info";
      case "Degraded": return "error";
      default: return "warning";
    }
  };

  return (
    <div className="space-y-6">
      {/* Rollout Status */}
      {detail.rollout && (
        <div className="flex items-center gap-3 p-4 bg-[var(--background)] rounded-lg">
          <Activity className="w-5 h-5 text-primary" />
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <StatusBadge status={detail.rollout.phase} type={getRolloutBadgeType(detail.rollout.phase)} />
              {detail.rollout.badges?.map((badge, i) => (
                <StatusBadge key={i} status={badge} type="info" />
              ))}
            </div>
            {detail.rollout.message && (
              <p className="text-sm text-muted mt-1">{detail.rollout.message}</p>
            )}
          </div>
        </div>
      )}

      {/* 副本状态 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">副本状态</h3>
        <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
          {[
            { label: "期望", value: detail.replicas, color: "text-default" },
            { label: "就绪", value: detail.ready, color: "text-green-500" },
            { label: "当前", value: detail.current, color: "text-blue-500" },
            { label: "已更新", value: detail.updated, color: "text-purple-500" },
            { label: "可用", value: detail.available, color: "text-cyan-500" },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3 text-center">
              <div className={`text-2xl font-bold ${item.color}`}>{item.value}</div>
              <div className="text-xs text-muted mt-1">{item.label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">基本信息</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: "名称", value: detail.name },
            { label: "命名空间", value: detail.namespace },
            { label: "Headless Service", value: detail.serviceName || "-" },
            { label: "Age", value: detail.age || "-" },
            { label: "Pod 管理策略", value: detail.spec.podManagementPolicy || "OrderedReady" },
            { label: "创建时间", value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium truncate">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Revision 信息 */}
      {(detail.status.currentRevision || detail.status.updateRevision) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Revision</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">当前 Revision</div>
              <div className="text-sm font-mono text-default truncate">
                {detail.status.currentRevision || "-"}
              </div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">更新 Revision</div>
              <div className="text-sm font-mono text-default truncate">
                {detail.status.updateRevision || "-"}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Conditions */}
      {detail.conditions && detail.conditions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Conditions</h3>
          <div className="space-y-2">
            {detail.conditions.map((c, i) => (
              <div key={i} className="flex items-start gap-3 p-3 bg-[var(--background)] rounded-lg">
                {c.status === "True" ? (
                  <CheckCircle className="w-4 h-4 text-green-500 mt-0.5" />
                ) : (
                  <AlertTriangle className="w-4 h-4 text-yellow-500 mt-0.5" />
                )}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-default">{c.type}</span>
                    <StatusBadge status={c.status} type={c.status === "True" ? "success" : "warning"} />
                  </div>
                  {c.message && (
                    <p className="text-sm text-muted mt-1 break-words">{c.message}</p>
                  )}
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
function ContainersTab({ detail }: { detail: StatefulSetDetail }) {
  const containers = detail.template?.containers || [];

  if (containers.length === 0) {
    return <div className="text-center py-8 text-muted">暂无容器信息</div>;
  }

  return (
    <div className="space-y-4">
      {containers.map((c, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium text-default">{c.name}</h4>
            {c.imagePullPolicy && (
              <StatusBadge status={c.imagePullPolicy} type="info" />
            )}
          </div>

          <div className="space-y-3">
            {/* 镜像 */}
            <div>
              <span className="text-xs text-muted">镜像: </span>
              <span className="text-sm font-mono text-default break-all">{c.image}</span>
            </div>

            {/* 端口 */}
            {c.ports && c.ports.length > 0 && (
              <div>
                <span className="text-xs text-muted block mb-1">端口:</span>
                <div className="flex flex-wrap gap-2">
                  {c.ports.map((p, j) => (
                    <span key={j} className="px-2 py-1 bg-[var(--card-background)] rounded text-xs font-mono">
                      {p.name ? `${p.name}:` : ""}{p.containerPort}/{p.protocol || "TCP"}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* 资源 */}
            {(c.requests || c.limits) && (
              <div className="grid grid-cols-2 gap-3">
                {c.requests && Object.keys(c.requests).length > 0 && (
                  <div>
                    <span className="text-xs text-muted block mb-1">Requests:</span>
                    <div className="text-sm font-mono">
                      {Object.entries(c.requests).map(([k, v]) => (
                        <div key={k}>{k}: {v}</div>
                      ))}
                    </div>
                  </div>
                )}
                {c.limits && Object.keys(c.limits).length > 0 && (
                  <div>
                    <span className="text-xs text-muted block mb-1">Limits:</span>
                    <div className="text-sm font-mono">
                      {Object.entries(c.limits).map(([k, v]) => (
                        <div key={k}>{k}: {v}</div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Probes */}
            {(c.livenessProbe || c.readinessProbe || c.startupProbe) && (
              <div className="grid grid-cols-3 gap-2 mt-2">
                {c.livenessProbe && (
                  <div className="p-2 bg-[var(--card-background)] rounded">
                    <div className="text-xs text-green-500 mb-1">Liveness</div>
                    <div className="text-xs text-muted">{c.livenessProbe.type}</div>
                  </div>
                )}
                {c.readinessProbe && (
                  <div className="p-2 bg-[var(--card-background)] rounded">
                    <div className="text-xs text-blue-500 mb-1">Readiness</div>
                    <div className="text-xs text-muted">{c.readinessProbe.type}</div>
                  </div>
                )}
                {c.startupProbe && (
                  <div className="p-2 bg-[var(--card-background)] rounded">
                    <div className="text-xs text-purple-500 mb-1">Startup</div>
                    <div className="text-xs text-muted">{c.startupProbe.type}</div>
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

// 策略 Tab
function StrategyTab({ detail }: { detail: StatefulSetDetail }) {
  const strategy = detail.spec.updateStrategy;

  return (
    <div className="space-y-6">
      {/* 更新策略 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">更新策略</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">策略类型</div>
            <div className="text-sm font-medium text-default">
              {strategy?.type || "RollingUpdate"}
            </div>
          </div>
          {strategy?.partition !== undefined && (
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">Partition</div>
              <div className="text-sm font-medium text-default">{strategy.partition}</div>
            </div>
          )}
          {strategy?.maxUnavailable && (
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">Max Unavailable</div>
              <div className="text-sm font-medium text-default">{strategy.maxUnavailable}</div>
            </div>
          )}
        </div>
      </div>

      {/* Pod 管理策略 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Pod 管理</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Pod 管理策略</div>
            <div className="text-sm font-medium text-default">
              {detail.spec.podManagementPolicy || "OrderedReady"}
            </div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Min Ready Seconds</div>
            <div className="text-sm font-medium text-default">
              {detail.spec.minReadySeconds || 0}
            </div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Revision History Limit</div>
            <div className="text-sm font-medium text-default">
              {detail.spec.revisionHistoryLimit ?? 10}
            </div>
          </div>
        </div>
      </div>

      {/* PVC 保留策略 */}
      {detail.spec.pvcRetentionPolicy && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">PVC 保留策略</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">删除时</div>
              <div className="text-sm font-medium text-default">
                {detail.spec.pvcRetentionPolicy.whenDeleted || "Retain"}
              </div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">缩容时</div>
              <div className="text-sm font-medium text-default">
                {detail.spec.pvcRetentionPolicy.whenScaled || "Retain"}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 调度信息 */}
      {(detail.template?.nodeSelector || detail.template?.tolerations?.length) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">调度</h3>
          <div className="space-y-3">
            {detail.template.nodeSelector && Object.keys(detail.template.nodeSelector).length > 0 && (
              <div className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-xs text-muted mb-2">Node Selector</div>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(detail.template.nodeSelector).map(([k, v]) => (
                    <span key={k} className="px-2 py-1 bg-[var(--card-background)] rounded text-xs font-mono">
                      {k}={v}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {detail.template.tolerations && detail.template.tolerations.length > 0 && (
              <div className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-xs text-muted mb-2">Tolerations ({detail.template.tolerations.length})</div>
                <div className="space-y-1">
                  {detail.template.tolerations.slice(0, 5).map((t, i) => (
                    <div key={i} className="text-xs font-mono text-default">
                      {t.key || "*"} {t.operator || "Equal"} {t.value || ""} : {t.effect || "All"}
                    </div>
                  ))}
                  {detail.template.tolerations.length > 5 && (
                    <div className="text-xs text-muted">... 还有 {detail.template.tolerations.length - 5} 个</div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// 存储 Tab
function StorageTab({ detail }: { detail: StatefulSetDetail }) {
  const vcts = detail.spec.volumeClaimTemplates || [];
  const volumes = detail.template?.volumes || [];

  return (
    <div className="space-y-6">
      {/* Volume Claim Templates */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">
          Volume Claim Templates ({vcts.length})
        </h3>
        {vcts.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">
            无 PVC 模板
          </div>
        ) : (
          <div className="space-y-3">
            {vcts.map((vct, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-4">
                <div className="flex items-center justify-between mb-2">
                  <h4 className="font-medium text-default">{vct.name}</h4>
                  {vct.storage && (
                    <span className="px-2 py-1 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-xs rounded">
                      {vct.storage}
                    </span>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-3 text-sm">
                  {vct.storageClass && (
                    <div>
                      <span className="text-muted">StorageClass: </span>
                      <span className="text-default">{vct.storageClass}</span>
                    </div>
                  )}
                  {vct.accessModes && vct.accessModes.length > 0 && (
                    <div>
                      <span className="text-muted">Access Modes: </span>
                      <span className="text-default">{vct.accessModes.join(", ")}</span>
                    </div>
                  )}
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Volumes */}
      {volumes.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">
            Volumes ({volumes.length})
          </h3>
          <div className="space-y-2">
            {volumes.map((v, i) => (
              <div key={i} className="flex items-center justify-between p-3 bg-[var(--background)] rounded-lg">
                <div>
                  <span className="font-medium text-default">{v.name}</span>
                  <span className="text-xs text-muted ml-2">({v.type})</span>
                </div>
                {v.source && (
                  <span className="text-sm text-muted truncate max-w-[200px]">{v.source}</span>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// 标签 Tab
function LabelsTab({ detail }: { detail: StatefulSetDetail }) {
  const labels = Object.entries(detail.labels || {});
  const annotations = Object.entries(detail.annotations || {});

  return (
    <div className="space-y-6">
      {/* Labels */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Labels ({labels.length})</h3>
        {labels.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">无标签</div>
        ) : (
          <div className="space-y-2">
            {labels.map(([key, value]) => (
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
        <h3 className="text-sm font-semibold text-default mb-3">Annotations ({annotations.length})</h3>
        {annotations.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">无注解</div>
        ) : (
          <div className="space-y-2">
            {annotations.map(([key, value]) => (
              <div key={key} className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-sm font-mono text-primary break-all mb-1">{key}</div>
                <div className="text-sm text-muted break-all">{value || '""'}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
