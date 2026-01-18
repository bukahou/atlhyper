"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge, ConfirmDialog } from "@/components/common";
import { getDeploymentDetail, scaleDeployment, updateDeploymentImage } from "@/api/deployment";
import { getCurrentClusterId } from "@/config/cluster";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useI18n } from "@/i18n/context";
import type { DeploymentDetail, DeploymentContainer, DeploymentReplicaSet, TolerationSpec, ProbeSpec } from "@/types/cluster";
import {
  Layers,
  Box,
  Tag,
  Settings,
  Save,
  X,
  Edit2,
  Plus,
  Minus,
  GitBranch,
  Calendar,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Pause,
  Activity,
  Server,
} from "lucide-react";

interface DeploymentDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  deploymentName: string;
  onUpdated?: () => void;
}

type TabType = "overview" | "containers" | "strategy" | "scheduling" | "replicasets" | "labels";

export function DeploymentDetailModal({
  isOpen,
  onClose,
  namespace,
  deploymentName,
  onUpdated,
}: DeploymentDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<DeploymentDetail | null>(null);
  const requireAuth = useRequireAuth();
  const { t } = useI18n();

  // 编辑状态
  const [editingImage, setEditingImage] = useState<{ containerName: string; oldImage: string; newImage: string } | null>(null);
  const [editingReplicas, setEditingReplicas] = useState<number | null>(null);
  const [saving, setSaving] = useState(false);
  const [confirmAction, setConfirmAction] = useState<"image" | "scale" | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!deploymentName || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getDeploymentDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Name: deploymentName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.deployment.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [namespace, deploymentName]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
      setEditingImage(null);
      setEditingReplicas(null);
    }
  }, [isOpen, fetchDetail]);

  // 更新镜像
  const handleUpdateImage = async () => {
    if (!editingImage || !detail) return;
    setSaving(true);
    try {
      await updateDeploymentImage({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Name: deploymentName,
        Kind: "Deployment",
        NewImage: editingImage.newImage,
        OldImage: editingImage.oldImage,
      });
      setEditingImage(null);
      setConfirmAction(null);
      setTimeout(() => {
        fetchDetail();
        onUpdated?.();
      }, 2000);
    } catch (err) {
      console.error("Update image failed:", err);
    } finally {
      setSaving(false);
    }
  };

  // 更新副本数
  const handleUpdateReplicas = async () => {
    if (editingReplicas === null || !detail) return;
    setSaving(true);
    try {
      await scaleDeployment({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Name: deploymentName,
        Kind: "Deployment",
        Replicas: editingReplicas,
      });
      setEditingReplicas(null);
      setConfirmAction(null);
      setTimeout(() => {
        fetchDetail();
        onUpdated?.();
      }, 2000);
    } catch (err) {
      console.error("Scale failed:", err);
    } finally {
      setSaving(false);
    }
  };

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.deployment.overview, icon: <Layers className="w-4 h-4" /> },
    { key: "containers", label: t.deployment.containers, icon: <Box className="w-4 h-4" /> },
    { key: "strategy", label: t.deployment.strategy, icon: <Settings className="w-4 h-4" /> },
    { key: "scheduling", label: t.deployment.scheduling, icon: <Server className="w-4 h-4" /> },
    { key: "replicasets", label: t.deployment.replicaSets, icon: <GitBranch className="w-4 h-4" /> },
    { key: "labels", label: t.deployment.labels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <>
      <Modal isOpen={isOpen} onClose={onClose} title={`Deployment: ${deploymentName}`} size="xl">
        {loading ? (
          <div className="py-12">
            <LoadingSpinner />
          </div>
        ) : error ? (
          <div className="p-6 text-center text-red-500">{error}</div>
        ) : detail ? (
          <div className="flex flex-col h-[70vh]">
            {/* Tabs */}
            <div className="flex border-b border-[var(--border-color)] px-4 shrink-0 overflow-x-auto">
              {tabs.map((tab) => (
                <button
                  key={tab.key}
                  onClick={() => setActiveTab(tab.key)}
                  className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors whitespace-nowrap ${
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
              {activeTab === "overview" && (
                <OverviewTab
                  detail={detail}
                  editingReplicas={editingReplicas}
                  onStartEdit={() => requireAuth(() => setEditingReplicas(detail.spec?.replicas ?? detail.replicas))}
                  onReplicasChange={setEditingReplicas}
                  onCancelEdit={() => setEditingReplicas(null)}
                  onSave={() => setConfirmAction("scale")}
                  t={t}
                />
              )}
              {activeTab === "containers" && (
                <ContainersTab
                  containers={detail.template?.containers || []}
                  editingImage={editingImage}
                  onEditImage={(containerName, oldImage) =>
                    requireAuth(() => setEditingImage({ containerName, oldImage, newImage: oldImage }))
                  }
                  onImageChange={(newImage) =>
                    setEditingImage((prev) => (prev ? { ...prev, newImage } : null))
                  }
                  onCancelEdit={() => setEditingImage(null)}
                  onSaveImage={() => setConfirmAction("image")}
                  t={t}
                />
              )}
              {activeTab === "strategy" && <StrategyTab detail={detail} t={t} />}
              {activeTab === "scheduling" && <SchedulingTab detail={detail} t={t} />}
              {activeTab === "replicasets" && <ReplicaSetsTab replicaSets={detail.replicaSets || []} t={t} />}
              {activeTab === "labels" && (
                <LabelsTab labels={detail.labels || {}} annotations={detail.annotations || {}} t={t} />
              )}
            </div>
          </div>
        ) : null}
      </Modal>

      {/* 镜像更新确认 */}
      <ConfirmDialog
        isOpen={confirmAction === "image"}
        onClose={() => setConfirmAction(null)}
        onConfirm={handleUpdateImage}
        title={t.deployment.confirmUpdateImage}
        message={
          editingImage
            ? t.deployment.confirmUpdateImageMessage
                .replace("{containerName}", editingImage.containerName)
                .replace("{oldImage}", editingImage.oldImage)
                .replace("{newImage}", editingImage.newImage)
            : ""
        }
        confirmText={t.common.update}
        cancelText={t.common.cancel}
        loading={saving}
        variant="warning"
      />

      {/* 扩缩容确认 */}
      <ConfirmDialog
        isOpen={confirmAction === "scale"}
        onClose={() => setConfirmAction(null)}
        onConfirm={handleUpdateReplicas}
        title={t.deployment.confirmScale}
        message={
          detail && editingReplicas !== null
            ? t.deployment.confirmScaleMessage
                .replace("{from}", String(detail.spec?.replicas ?? detail.replicas))
                .replace("{to}", String(editingReplicas))
            : ""
        }
        confirmText={t.common.confirm}
        cancelText={t.common.cancel}
        loading={saving}
        variant="info"
      />
    </>
  );
}

// 概览 Tab
function OverviewTab({
  detail,
  editingReplicas,
  onStartEdit,
  onReplicasChange,
  onCancelEdit,
  onSave,
  t,
}: {
  detail: DeploymentDetail;
  editingReplicas: number | null;
  onStartEdit: () => void;
  onReplicasChange: (n: number) => void;
  onCancelEdit: () => void;
  onSave: () => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  const isEditing = editingReplicas !== null;
  const replicas = editingReplicas ?? detail.replicas;

  return (
    <div className="space-y-6">
      {/* Rollout 状态徽标 */}
      {detail.rollout && (
        <div className="flex items-center gap-2 flex-wrap">
          {detail.rollout.badges?.map((badge, i) => (
            <RolloutBadge key={i} badge={badge} />
          ))}
          {detail.paused && (
            <span className="inline-flex items-center gap-1 px-2 py-1 bg-yellow-100 dark:bg-yellow-900/30 text-yellow-600 dark:text-yellow-400 text-xs rounded">
              <Pause className="w-3 h-3" /> {t.deployment.paused}
            </span>
          )}
        </div>
      )}

      {/* 副本状态卡片 */}
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-[var(--background)] rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-default">{detail.replicas}</div>
          <div className="text-xs text-muted mt-1">{t.deployment.desired}</div>
        </div>
        <div className="bg-[var(--background)] rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-green-500">{detail.ready}</div>
          <div className="text-xs text-muted mt-1">{t.deployment.ready}</div>
        </div>
        <div className="bg-[var(--background)] rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-blue-500">{detail.available}</div>
          <div className="text-xs text-muted mt-1">{t.deployment.available}</div>
        </div>
        <div className="bg-[var(--background)] rounded-lg p-4 text-center">
          <div className="text-2xl font-bold text-orange-500">{detail.updated}</div>
          <div className="text-xs text-muted mt-1">{t.deployment.updated}</div>
        </div>
      </div>

      {/* 扩缩容 */}
      <div className="bg-[var(--background)] rounded-lg p-4">
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.adjustReplicas}</h3>
        {isEditing ? (
          <div className="flex items-center gap-4">
            <button
              onClick={() => onReplicasChange(Math.max(0, replicas - 1))}
              className="p-2 bg-card hover:bg-[var(--border-color)] rounded-lg"
            >
              <Minus className="w-5 h-5" />
            </button>
            <input
              type="number"
              min="0"
              value={replicas}
              onChange={(e) => onReplicasChange(Math.max(0, parseInt(e.target.value) || 0))}
              className="w-20 px-3 py-2 text-center text-xl font-bold bg-card border border-[var(--border-color)] rounded-lg"
            />
            <button
              onClick={() => onReplicasChange(replicas + 1)}
              className="p-2 bg-card hover:bg-[var(--border-color)] rounded-lg"
            >
              <Plus className="w-5 h-5" />
            </button>
            <button
              onClick={onSave}
              disabled={replicas === detail.replicas}
              className="px-3 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50"
            >
              <Save className="w-4 h-4" />
            </button>
            <button onClick={onCancelEdit} className="px-3 py-2 hover-bg rounded-lg">
              <X className="w-4 h-4 text-muted" />
            </button>
          </div>
        ) : (
          <button
            onClick={onStartEdit}
            className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 flex items-center gap-2"
          >
            <Edit2 className="w-4 h-4" />
            {t.deployment.adjustReplicas}
          </button>
        )}
      </div>

      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.basicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <InfoCard label={t.common.name} value={detail.name} />
          <InfoCard label={t.common.namespace} value={detail.namespace} />
          <InfoCard label={t.deployment.strategy} value={detail.strategy || "-"} />
          <InfoCard label={t.deployment.age} value={detail.age || "-"} />
          <InfoCard label={t.deployment.selector} value={detail.selector || "-"} />
          <InfoCard label={t.common.createdAt} value={detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-"} />
        </div>
      </div>

      {/* Conditions */}
      {detail.conditions && detail.conditions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.conditions}</h3>
          <div className="space-y-2">
            {detail.conditions.map((cond, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-3 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  {cond.status === "True" ? (
                    <CheckCircle className="w-4 h-4 text-green-500" />
                  ) : cond.status === "False" ? (
                    <XCircle className="w-4 h-4 text-gray-400" />
                  ) : (
                    <AlertTriangle className="w-4 h-4 text-yellow-500" />
                  )}
                  <span className="font-medium text-default">{cond.type}</span>
                  {cond.reason && <span className="text-sm text-muted">({cond.reason})</span>}
                </div>
                <StatusBadge status={cond.status} type={cond.status === "True" ? "success" : "warning"} />
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// Rollout 徽标组件
function RolloutBadge({ badge }: { badge: string }) {
  const config: Record<string, { bg: string; text: string; icon: React.ReactNode }> = {
    Available: { bg: "bg-green-100 dark:bg-green-900/30", text: "text-green-600 dark:text-green-400", icon: <CheckCircle className="w-3 h-3" /> },
    Progressing: { bg: "bg-blue-100 dark:bg-blue-900/30", text: "text-blue-600 dark:text-blue-400", icon: <Activity className="w-3 h-3" /> },
    Failed: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-600 dark:text-red-400", icon: <XCircle className="w-3 h-3" /> },
    ReplicaFailure: { bg: "bg-red-100 dark:bg-red-900/30", text: "text-red-600 dark:text-red-400", icon: <AlertTriangle className="w-3 h-3" /> },
    Paused: { bg: "bg-yellow-100 dark:bg-yellow-900/30", text: "text-yellow-600 dark:text-yellow-400", icon: <Pause className="w-3 h-3" /> },
  };
  const c = config[badge] || { bg: "bg-gray-100 dark:bg-gray-800", text: "text-gray-600 dark:text-gray-400", icon: null };
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-1 ${c.bg} ${c.text} text-xs rounded`}>
      {c.icon} {badge}
    </span>
  );
}

// 信息卡片组件
function InfoCard({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="bg-[var(--background)] rounded-lg p-3">
      <div className="text-xs text-muted mb-1">{label}</div>
      <div className="text-sm text-default font-medium break-all">{value}</div>
    </div>
  );
}

// 容器 Tab
function ContainersTab({
  containers,
  editingImage,
  onEditImage,
  onImageChange,
  onCancelEdit,
  onSaveImage,
  t,
}: {
  containers: DeploymentContainer[];
  editingImage: { containerName: string; oldImage: string; newImage: string } | null;
  onEditImage: (containerName: string, oldImage: string) => void;
  onImageChange: (newImage: string) => void;
  onCancelEdit: () => void;
  onSaveImage: () => void;
  t: ReturnType<typeof useI18n>["t"];
}) {
  if (!containers || containers.length === 0) {
    return <div className="text-center py-8 text-muted">{t.deployment.noContainers}</div>;
  }

  return (
    <div className="space-y-4">
      {containers.map((container) => (
        <div key={container.name} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium text-default">{container.name}</h4>
            {container.imagePullPolicy && <StatusBadge status={container.imagePullPolicy} type="info" />}
          </div>

          {/* 镜像 */}
          <div className="mb-4">
            <div className="text-xs text-muted mb-1">{t.deployment.image}</div>
            {editingImage?.containerName === container.name ? (
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={editingImage.newImage}
                  onChange={(e) => onImageChange(e.target.value)}
                  className="flex-1 px-3 py-2 bg-card border border-[var(--border-color)] rounded-lg text-sm"
                />
                <button
                  onClick={onSaveImage}
                  disabled={editingImage.newImage === editingImage.oldImage}
                  className="p-2 bg-primary text-white rounded-lg disabled:opacity-50"
                >
                  <Save className="w-4 h-4" />
                </button>
                <button onClick={onCancelEdit} className="p-2 hover-bg rounded-lg">
                  <X className="w-4 h-4 text-muted" />
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <span className="text-sm font-mono text-default break-all">{container.image}</span>
                <button onClick={() => onEditImage(container.name, container.image)} className="p-1.5 hover-bg rounded-lg shrink-0">
                  <Edit2 className="w-3.5 h-3.5 text-muted" />
                </button>
              </div>
            )}
          </div>

          {/* 端口 */}
          {container.ports && container.ports.length > 0 && (
            <div className="mb-3">
              <div className="text-xs text-muted mb-1">{t.deployment.ports}</div>
              <div className="flex flex-wrap gap-2">
                {container.ports.map((port, i) => (
                  <span key={i} className="px-2 py-1 bg-card rounded text-xs font-mono">
                    {port.containerPort}/{port.protocol || "TCP"}
                    {port.name && ` (${port.name})`}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* 资源限制 */}
          {(container.requests || container.limits) && (
            <div className="grid grid-cols-2 gap-4 mb-3">
              {container.requests && Object.keys(container.requests).length > 0 && (
                <div>
                  <div className="text-xs text-muted mb-1">Requests</div>
                  <div className="space-y-1">
                    {Object.entries(container.requests).map(([k, v]) => (
                      <div key={k} className="text-xs">
                        <span className="text-muted">{k}:</span> <span className="font-mono">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {container.limits && Object.keys(container.limits).length > 0 && (
                <div>
                  <div className="text-xs text-muted mb-1">Limits</div>
                  <div className="space-y-1">
                    {Object.entries(container.limits).map(([k, v]) => (
                      <div key={k} className="text-xs">
                        <span className="text-muted">{k}:</span> <span className="font-mono">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}

          {/* 探针 */}
          <ProbesDisplay
            liveness={container.livenessProbe}
            readiness={container.readinessProbe}
            startup={container.startupProbe}
            t={t}
          />

          {/* 环境变量 */}
          {container.envs && container.envs.length > 0 && (
            <details className="mt-3">
              <summary className="text-xs text-muted cursor-pointer">{t.deployment.envVars} ({container.envs.length})</summary>
              <div className="mt-2 space-y-1">
                {container.envs.map((env, i) => (
                  <div key={i} className="text-xs font-mono bg-card px-2 py-1 rounded">
                    <span className="text-primary">{env.name}</span>
                    <span className="text-muted">=</span>
                    <span className="text-default">{env.value || env.valueFrom || '""'}</span>
                  </div>
                ))}
              </div>
            </details>
          )}

          {/* 挂载 */}
          {container.volumeMounts && container.volumeMounts.length > 0 && (
            <details className="mt-3">
              <summary className="text-xs text-muted cursor-pointer">{t.deployment.volumeMounts} ({container.volumeMounts.length})</summary>
              <div className="mt-2 space-y-1">
                {container.volumeMounts.map((vm, i) => (
                  <div key={i} className="text-xs font-mono bg-card px-2 py-1 rounded flex justify-between">
                    <span className="text-primary">{vm.name}</span>
                    <span className="text-default">{vm.mountPath}</span>
                    {vm.readOnly && <span className="text-muted">(ro)</span>}
                  </div>
                ))}
              </div>
            </details>
          )}
        </div>
      ))}
    </div>
  );
}

// 探针显示组件
function ProbesDisplay({ liveness, readiness, startup, t }: { liveness?: ProbeSpec; readiness?: ProbeSpec; startup?: ProbeSpec; t: ReturnType<typeof useI18n>["t"] }) {
  const probes = [
    { name: "Liveness", probe: liveness },
    { name: "Readiness", probe: readiness },
    { name: "Startup", probe: startup },
  ].filter((p) => p.probe);

  if (probes.length === 0) return null;

  return (
    <div className="mt-3">
      <div className="text-xs text-muted mb-2">{t.deployment.probes}</div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
        {probes.map(({ name, probe }) => (
          <div key={name} className="bg-card rounded p-2">
            <div className="text-xs font-medium text-default mb-1">{name}</div>
            <div className="text-xs text-muted">
              {probe!.type === "httpGet" && `HTTP GET ${probe!.path || "/"}:${probe!.port}`}
              {probe!.type === "tcpSocket" && `TCP :${probe!.port}`}
              {probe!.type === "exec" && `Exec: ${probe!.command}`}
            </div>
            <div className="text-xs text-muted mt-1">
              delay={probe!.initialDelaySeconds || 0}s period={probe!.periodSeconds || 10}s
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// 策略 Tab
function StrategyTab({ detail, t }: { detail: DeploymentDetail; t: ReturnType<typeof useI18n>["t"] }) {
  const spec = detail.spec;
  return (
    <div className="space-y-6">
      {/* 更新策略 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.updateStrategy}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <InfoCard label={t.deployment.strategyType} value={spec?.strategyType || detail.strategy || "-"} />
          <InfoCard label="MaxUnavailable" value={spec?.maxUnavailable || "-"} />
          <InfoCard label="MaxSurge" value={spec?.maxSurge || "-"} />
        </div>
      </div>

      {/* 其他配置 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.otherConfig}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <InfoCard label="MinReadySeconds" value={spec?.minReadySeconds ?? 0} />
          <InfoCard label="RevisionHistoryLimit" value={spec?.revisionHistoryLimit ?? 10} />
          <InfoCard label="ProgressDeadlineSeconds" value={spec?.progressDeadlineSeconds ?? 600} />
        </div>
      </div>

      {/* 选择器 */}
      {spec?.matchLabels && Object.keys(spec.matchLabels).length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.selector}</h3>
          <div className="space-y-2">
            {Object.entries(spec.matchLabels).map(([k, v]) => (
              <div key={k} className="bg-[var(--background)] rounded-lg p-3 flex items-start gap-2">
                <span className="text-sm font-mono text-primary">{k}</span>
                <span className="text-muted">=</span>
                <span className="text-sm font-mono text-default">{v}</span>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// 调度 Tab
function SchedulingTab({ detail, t }: { detail: DeploymentDetail; t: ReturnType<typeof useI18n>["t"] }) {
  const template = detail.template;
  return (
    <div className="space-y-6">
      {/* 基本调度配置 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.schedulingConfig}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <InfoCard label="ServiceAccount" value={template?.serviceAccountName || "default"} />
          <InfoCard label="RuntimeClass" value={template?.runtimeClassName || "-"} />
          <InfoCard label="DNSPolicy" value={template?.dnsPolicy || "ClusterFirst"} />
          <InfoCard label={t.deployment.hostNetwork} value={template?.hostNetwork ? t.common.yes : t.common.no} />
        </div>
      </div>

      {/* NodeSelector */}
      {template?.nodeSelector && Object.keys(template.nodeSelector).length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">NodeSelector</h3>
          <div className="space-y-2">
            {Object.entries(template.nodeSelector).map(([k, v]) => (
              <div key={k} className="bg-[var(--background)] rounded-lg p-3 flex items-start gap-2">
                <span className="text-sm font-mono text-primary">{k}</span>
                <span className="text-muted">=</span>
                <span className="text-sm font-mono text-default">{v}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tolerations */}
      {template?.tolerations && template.tolerations.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Tolerations</h3>
          <div className="space-y-2">
            {template.tolerations.map((tol: TolerationSpec, i: number) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-3">
                <div className="flex items-center gap-2 flex-wrap">
                  {tol.key && <span className="text-sm font-mono text-primary">{tol.key}</span>}
                  {tol.operator && <span className="text-xs text-muted">{tol.operator}</span>}
                  {tol.value && <span className="text-sm font-mono text-default">{tol.value}</span>}
                  {tol.effect && (
                    <span className="px-2 py-0.5 bg-card text-xs rounded">{tol.effect}</span>
                  )}
                  {tol.tolerationSeconds !== undefined && (
                    <span className="text-xs text-muted">({tol.tolerationSeconds}s)</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Affinity */}
      {template?.affinity && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Affinity</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {template.affinity.nodeAffinity && <InfoCard label="NodeAffinity" value={template.affinity.nodeAffinity} />}
            {template.affinity.podAffinity && <InfoCard label="PodAffinity" value={template.affinity.podAffinity} />}
            {template.affinity.podAntiAffinity && <InfoCard label="PodAntiAffinity" value={template.affinity.podAntiAffinity} />}
          </div>
        </div>
      )}

      {/* ImagePullSecrets */}
      {template?.imagePullSecrets && template.imagePullSecrets.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">ImagePullSecrets</h3>
          <div className="flex flex-wrap gap-2">
            {template.imagePullSecrets.map((s, i) => (
              <span key={i} className="px-2 py-1 bg-[var(--background)] text-sm font-mono rounded">{s}</span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// 副本集 Tab
function ReplicaSetsTab({ replicaSets, t }: { replicaSets: DeploymentReplicaSet[]; t: ReturnType<typeof useI18n>["t"] }) {
  if (replicaSets.length === 0) {
    return <div className="text-center py-8 text-muted">{t.deployment.noReplicaSets}</div>;
  }

  // 按 revision 排序，最新的在前
  const sorted = [...replicaSets].sort((a, b) => {
    const ra = parseInt(a.revision || "0", 10);
    const rb = parseInt(b.revision || "0", 10);
    return rb - ra;
  });

  return (
    <div className="space-y-3">
      {sorted.map((rs, i) => (
        <div
          key={rs.name}
          className={`bg-[var(--background)] rounded-lg p-4 ${i === 0 ? "ring-2 ring-primary" : ""}`}
        >
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <span className="font-medium text-default">{rs.name}</span>
              {rs.revision && (
                <span className="px-2 py-0.5 bg-card text-xs rounded">Rev {rs.revision}</span>
              )}
              {i === 0 && (
                <span className="px-2 py-0.5 bg-primary/10 text-primary text-xs rounded">{t.deployment.current}</span>
              )}
            </div>
            <div className="flex items-center gap-2">
              <Calendar className="w-4 h-4 text-muted" />
              <span className="text-sm text-muted">{rs.createdAt ? new Date(rs.createdAt).toLocaleString() : "-"}</span>
            </div>
          </div>
          <div className="grid grid-cols-3 gap-4 text-center">
            <div>
              <div className="text-lg font-bold text-default">{rs.replicas}</div>
              <div className="text-xs text-muted">{t.deployment.desired}</div>
            </div>
            <div>
              <div className="text-lg font-bold text-green-500">{rs.ready}</div>
              <div className="text-xs text-muted">{t.deployment.ready}</div>
            </div>
            <div>
              <div className="text-lg font-bold text-blue-500">{rs.available}</div>
              <div className="text-xs text-muted">{t.deployment.available}</div>
            </div>
          </div>
          {rs.image && (
            <div className="mt-2 text-xs font-mono text-muted truncate">{rs.image}</div>
          )}
        </div>
      ))}
    </div>
  );
}

// 标签 Tab
function LabelsTab({ labels, annotations, t }: { labels: Record<string, string>; annotations: Record<string, string>; t: ReturnType<typeof useI18n>["t"] }) {
  const labelEntries = Object.entries(labels);
  const annoEntries = Object.entries(annotations);

  return (
    <div className="space-y-6">
      {/* Labels */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.labels} ({labelEntries.length})</h3>
        {labelEntries.length === 0 ? (
          <div className="text-center py-4 text-muted">{t.deployment.noLabels}</div>
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
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.annotations} ({annoEntries.length})</h3>
        {annoEntries.length === 0 ? (
          <div className="text-center py-4 text-muted">{t.deployment.noAnnotations}</div>
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
