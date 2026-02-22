"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { ConfirmDialog } from "@/components/common";
import { getDeploymentDetail, scaleDeployment, updateDeploymentImage } from "@/datasource/cluster";
import { getCurrentClusterId } from "@/config/cluster";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useI18n } from "@/i18n/context";
import type { DeploymentDetail } from "@/types/cluster";
import { Layers, Box, Tag, Settings, GitBranch, Server } from "lucide-react";

// Tab 组件
import {
  OverviewTab,
  ContainersTab,
  StrategyTab,
  SchedulingTab,
  ReplicaSetsTab,
  LabelsTab,
} from "./tabs";

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
  }, [namespace, deploymentName, t.deployment.loadFailed]);

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
        ContainerName: editingImage.containerName,
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
      <Drawer isOpen={isOpen} onClose={onClose} title={`Deployment: ${deploymentName}`} size="xl">
        {loading ? (
          <div className="py-12">
            <LoadingSpinner />
          </div>
        ) : error ? (
          <div className="p-6 text-center text-red-500">{error}</div>
        ) : detail ? (
          <div className="flex flex-col h-full">
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
      </Drawer>

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
