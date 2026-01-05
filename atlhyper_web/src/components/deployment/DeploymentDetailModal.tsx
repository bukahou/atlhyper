"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge, ConfirmDialog } from "@/components/common";
import { getDeploymentDetail, scaleDeployment, updateDeploymentImage } from "@/api/deployment";
import { getCurrentClusterId } from "@/config/cluster";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import type { DeploymentDetail, DeploymentContainer } from "@/types/cluster";
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
} from "lucide-react";

interface DeploymentDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  deploymentName: string;
  onUpdated?: () => void;
}

type TabType = "overview" | "containers" | "scale" | "labels";

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
      setError(err instanceof Error ? err.message : "加载失败");
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
      // 延迟刷新
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
      // 延迟刷新
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
    { key: "overview", label: "概览", icon: <Layers className="w-4 h-4" /> },
    { key: "containers", label: "容器", icon: <Box className="w-4 h-4" /> },
    { key: "scale", label: "扩缩容", icon: <Settings className="w-4 h-4" /> },
    { key: "labels", label: "标签", icon: <Tag className="w-4 h-4" /> },
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
                />
              )}
              {activeTab === "scale" && (
                <ScaleTab
                  currentReplicas={detail.spec?.replicas ?? detail.replicas}
                  readyReplicas={detail.ready}
                  availableReplicas={detail.available}
                  editingReplicas={editingReplicas}
                  onStartEdit={() => requireAuth(() => setEditingReplicas(detail.spec?.replicas ?? detail.replicas))}
                  onReplicasChange={setEditingReplicas}
                  onCancelEdit={() => setEditingReplicas(null)}
                  onSave={() => setConfirmAction("scale")}
                />
              )}
              {activeTab === "labels" && (
                <LabelsTab labels={detail.labels || {}} annotations={detail.annotations || {}} />
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
        title="确认更新镜像"
        message={
          editingImage
            ? `确定要将容器 "${editingImage.containerName}" 的镜像从 "${editingImage.oldImage}" 更新为 "${editingImage.newImage}" 吗？这将触发滚动更新。`
            : ""
        }
        confirmText="更新"
        cancelText="取消"
        loading={saving}
        variant="warning"
      />

      {/* 扩缩容确认 */}
      <ConfirmDialog
        isOpen={confirmAction === "scale"}
        onClose={() => setConfirmAction(null)}
        onConfirm={handleUpdateReplicas}
        title="确认扩缩容"
        message={
          detail && editingReplicas !== null
            ? `确定要将副本数从 ${detail.spec?.replicas ?? detail.replicas} 调整为 ${editingReplicas} 吗？`
            : ""
        }
        confirmText="确认"
        cancelText="取消"
        loading={saving}
        variant="info"
      />
    </>
  );
}

// 概览 Tab
function OverviewTab({ detail }: { detail: DeploymentDetail }) {
  const infoItems = [
    { label: "名称", value: detail.name },
    { label: "命名空间", value: detail.namespace },
    { label: "副本", value: `${detail.ready}/${detail.replicas}` },
    { label: "策略", value: detail.strategy || detail.spec?.strategyType || "-" },
    { label: "Ready", value: detail.ready },
    { label: "Available", value: detail.available },
    { label: "暂停", value: detail.paused ? "是" : "否" },
    { label: "Age", value: detail.age || "-" },
    { label: "创建时间", value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
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

      {/* 策略详情 */}
      {(detail.spec?.maxUnavailable || detail.spec?.maxSurge) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">滚动更新策略</h3>
          <div className="grid grid-cols-2 gap-4">
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">MaxUnavailable</div>
              <div className="text-sm text-default font-medium">{detail.spec?.maxUnavailable || "-"}</div>
            </div>
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">MaxSurge</div>
              <div className="text-sm text-default font-medium">{detail.spec?.maxSurge || "-"}</div>
            </div>
          </div>
        </div>
      )}

      {/* Conditions */}
      {detail.conditions && detail.conditions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Conditions</h3>
          <div className="space-y-2">
            {detail.conditions.map((cond, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-3 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <StatusBadge status={cond.type} type={cond.status === "True" ? "success" : "warning"} />
                  {cond.reason && <span className="text-sm text-muted">{cond.reason}</span>}
                </div>
                <span className="text-xs text-muted">{cond.status}</span>
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
  editingImage,
  onEditImage,
  onImageChange,
  onCancelEdit,
  onSaveImage,
}: {
  containers: DeploymentContainer[];
  editingImage: { containerName: string; oldImage: string; newImage: string } | null;
  onEditImage: (containerName: string, oldImage: string) => void;
  onImageChange: (newImage: string) => void;
  onCancelEdit: () => void;
  onSaveImage: () => void;
}) {
  if (!containers || containers.length === 0) {
    return <div className="text-center py-8 text-muted">暂无容器信息</div>;
  }

  return (
    <div className="space-y-4">
      {containers.map((container) => (
        <div key={container.name} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium text-default">{container.name}</h4>
            {container.imagePullPolicy && (
              <StatusBadge status={container.imagePullPolicy} type="info" />
            )}
          </div>

          {/* 镜像 */}
          <div className="mb-4">
            <div className="text-xs text-muted mb-1">镜像</div>
            {editingImage?.containerName === container.name ? (
              <div className="flex items-center gap-2">
                <input
                  type="text"
                  value={editingImage.newImage}
                  onChange={(e) => onImageChange(e.target.value)}
                  className="flex-1 px-3 py-2 bg-card border border-[var(--border-color)] rounded-lg text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary"
                />
                <button
                  onClick={onSaveImage}
                  disabled={editingImage.newImage === editingImage.oldImage}
                  className="p-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed"
                  title="保存"
                >
                  <Save className="w-4 h-4" />
                </button>
                <button
                  onClick={onCancelEdit}
                  className="p-2 hover-bg rounded-lg"
                  title="取消"
                >
                  <X className="w-4 h-4 text-muted" />
                </button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <span className="text-sm font-mono text-default break-all">{container.image}</span>
                <button
                  onClick={() => onEditImage(container.name, container.image)}
                  className="p-1.5 hover-bg rounded-lg shrink-0"
                  title="编辑镜像"
                >
                  <Edit2 className="w-3.5 h-3.5 text-muted" />
                </button>
              </div>
            )}
          </div>

          {/* 端口 */}
          {container.ports && container.ports.length > 0 && (
            <div className="mb-3">
              <div className="text-xs text-muted mb-1">端口</div>
              <div className="flex flex-wrap gap-2">
                {container.ports.map((port, i) => (
                  <span key={i} className="px-2 py-1 bg-card rounded text-xs font-mono">
                    {port.containerPort}/{port.protocol}
                    {port.name && ` (${port.name})`}
                  </span>
                ))}
              </div>
            </div>
          )}

          {/* 资源限制 */}
          {(container.requests || container.limits) && (
            <div className="grid grid-cols-2 gap-4">
              {container.requests && Object.keys(container.requests).length > 0 && (
                <div>
                  <div className="text-xs text-muted mb-1">Requests</div>
                  <div className="space-y-1">
                    {Object.entries(container.requests).map(([k, v]) => (
                      <div key={k} className="text-xs">
                        <span className="text-muted">{k}:</span>{" "}
                        <span className="font-mono">{v}</span>
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
                        <span className="text-muted">{k}:</span>{" "}
                        <span className="font-mono">{v}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// 扩缩容 Tab
function ScaleTab({
  currentReplicas,
  readyReplicas,
  availableReplicas,
  editingReplicas,
  onStartEdit,
  onReplicasChange,
  onCancelEdit,
  onSave,
}: {
  currentReplicas: number;
  readyReplicas: number;
  availableReplicas: number;
  editingReplicas: number | null;
  onStartEdit: () => void;
  onReplicasChange: (replicas: number) => void;
  onCancelEdit: () => void;
  onSave: () => void;
}) {
  const isEditing = editingReplicas !== null;
  const replicas = editingReplicas ?? currentReplicas;

  return (
    <div className="space-y-6">
      {/* 当前状态 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">当前状态</h3>
        <div className="grid grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-default">{currentReplicas}</div>
            <div className="text-xs text-muted mt-1">期望副本</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-green-500">{readyReplicas}</div>
            <div className="text-xs text-muted mt-1">就绪</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-blue-500">{availableReplicas}</div>
            <div className="text-xs text-muted mt-1">可用</div>
          </div>
        </div>
      </div>

      {/* 调整副本数 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">调整副本数</h3>
        <div className="bg-[var(--background)] rounded-lg p-6">
          {isEditing ? (
            <div className="space-y-4">
              {/* 上下布局的调整控件 */}
              <div className="flex flex-col items-center gap-2">
                <button
                  onClick={() => onReplicasChange(replicas + 1)}
                  className="p-3 bg-card hover:bg-[var(--border-color)] rounded-lg transition-colors w-24"
                >
                  <Plus className="w-5 h-5 mx-auto" />
                </button>
                <input
                  type="number"
                  min="0"
                  value={replicas}
                  onChange={(e) => onReplicasChange(Math.max(0, parseInt(e.target.value) || 0))}
                  className="w-24 px-4 py-3 text-center text-2xl font-bold bg-card border border-[var(--border-color)] rounded-lg focus:outline-none focus:ring-1 focus:ring-primary"
                />
                <button
                  onClick={() => onReplicasChange(Math.max(0, replicas - 1))}
                  className="p-3 bg-card hover:bg-[var(--border-color)] rounded-lg transition-colors w-24"
                >
                  <Minus className="w-5 h-5 mx-auto" />
                </button>
              </div>

              {replicas !== currentReplicas && (
                <div className="text-center text-sm text-muted">
                  {replicas > currentReplicas
                    ? `将增加 ${replicas - currentReplicas} 个副本`
                    : `将减少 ${currentReplicas - replicas} 个副本`}
                </div>
              )}

              <div className="flex items-center justify-center gap-3">
                <button
                  onClick={onSave}
                  disabled={replicas === currentReplicas}
                  className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                >
                  <Save className="w-4 h-4" />
                  保存
                </button>
                <button
                  onClick={onCancelEdit}
                  className="px-4 py-2 bg-card hover:bg-[var(--border-color)] rounded-lg flex items-center gap-2"
                >
                  <X className="w-4 h-4" />
                  取消
                </button>
              </div>
            </div>
          ) : (
            <div className="text-center">
              <div className="text-4xl font-bold text-default mb-4">{currentReplicas}</div>
              <button
                onClick={onStartEdit}
                className="px-4 py-2 bg-primary text-white rounded-lg hover:bg-primary/90 flex items-center gap-2 mx-auto"
              >
                <Edit2 className="w-4 h-4" />
                调整副本数
              </button>
            </div>
          )}
        </div>
      </div>
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
