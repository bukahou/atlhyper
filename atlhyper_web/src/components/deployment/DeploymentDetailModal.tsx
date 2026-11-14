"use client";

import { useState } from "react";
import { useMutation, useQuery, type QueryKey } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import { ConfirmDialog } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useDelayedInvalidate } from "@/hooks/useDelayedInvalidate";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useI18n } from "@/i18n/context";
import { deploymentDetailQuery } from "@/queries/cluster";
import { scaleDeploymentMutation, updateDeploymentImageMutation } from "@/queries/cluster-mutations";
import { clusterKeys } from "@/queries/keys";
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
}

interface DeploymentDetailDrawerProps extends Omit<DeploymentDetailModalProps, "isOpen"> {
  /** 过一会儿让给定的查询重新取数（由外层持有，抽屉关掉后仍然生效） */
  invalidateAfterCommand: (queryKeys: readonly QueryKey[]) => void;
}

type TabType = "overview" | "containers" | "strategy" | "scheduling" | "replicasets" | "labels";

export function DeploymentDetailModal({ isOpen, onClose, namespace, deploymentName }: DeploymentDetailModalProps) {
  // 延时刷新挂在外层：操作完马上关掉抽屉，稍后那次列表刷新仍然要发生
  const invalidateAfterCommand = useDelayedInvalidate();

  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页与编辑中的草稿随关闭丢弃
  if (!isOpen) return null;
  return (
    <DeploymentDetailDrawer
      key={`${namespace}/${deploymentName}`}
      onClose={onClose}
      namespace={namespace}
      deploymentName={deploymentName}
      invalidateAfterCommand={invalidateAfterCommand}
    />
  );
}

function DeploymentDetailDrawer({
  onClose,
  namespace,
  deploymentName,
  invalidateAfterCommand,
}: DeploymentDetailDrawerProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const requireAuth = useRequireAuth();
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(deploymentDetailQuery(clusterId, { namespace, name: deploymentName }));
  const detail = query.data;
  const scale = useMutation(scaleDeploymentMutation(clusterId));
  const updateImage = useMutation(updateDeploymentImageMutation(clusterId));

  // 编辑状态
  const [editingImage, setEditingImage] = useState<{ containerName: string; oldImage: string; newImage: string } | null>(null);
  const [editingReplicas, setEditingReplicas] = useState<number | null>(null);
  const [confirmAction, setConfirmAction] = useState<"image" | "scale" | null>(null);

  // 指令由 Agent 异步执行：下发成功时列表与详情已刷新过一次，
  // 晚一点再刷新一次才看得到滚动更新后的状态
  const refreshAfterCommand = () => invalidateAfterCommand(clusterKeys.resourceScope("deployment"));

  // 更新镜像
  const handleUpdateImage = () => {
    if (!editingImage || !detail) return;
    updateImage.mutate(
      { namespace, name: deploymentName, ...editingImage },
      {
        onSuccess: () => {
          setEditingImage(null);
          setConfirmAction(null);
          refreshAfterCommand();
        },
        onError: (err) => console.error("Update image failed:", err),
      },
    );
  };

  // 更新副本数
  const handleUpdateReplicas = () => {
    if (editingReplicas === null || !detail) return;
    scale.mutate(
      { namespace, name: deploymentName, replicas: editingReplicas },
      {
        onSuccess: () => {
          setEditingReplicas(null);
          setConfirmAction(null);
          refreshAfterCommand();
        },
        onError: (err) => console.error("Scale failed:", err),
      },
    );
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
      <Drawer isOpen onClose={onClose} title={`Deployment: ${deploymentName}`} size="xl">
        <DetailQueryView query={query} errorFallback={t.deployment.loadFailed}>
          {(detail) => (
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
          )}
        </DetailQueryView>
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
        loading={updateImage.isPending}
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
        loading={scale.isPending}
        variant="info"
      />
    </>
  );
}
