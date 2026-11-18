"use client";

import { useState } from "react";
import { useMutation, useQuery, type QueryKey } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import { ConfirmDialog } from "@/components/common";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useDelayedInvalidate } from "@/hooks/useDelayedInvalidate";
import { useI18n } from "@/i18n/context";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { nodeDetailQuery } from "@/queries/cluster";
import { nodeSchedulingMutation } from "@/queries/cluster-mutations";
import { clusterKeys } from "@/queries/keys";
import { OverviewTab, ResourcesTab, ConditionsTab, TaintsTab, LabelsTab } from "./NodeDetailTabs";
import { Server, Cpu, CheckCircle, AlertTriangle, Tag, Shield, ShieldOff } from "lucide-react";

interface NodeDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
}

interface NodeDetailDrawerProps extends Omit<NodeDetailModalProps, "isOpen"> {
  /** 过一会儿让给定的查询重新取数（由外层持有，抽屉关掉后仍然生效） */
  invalidateAfterCommand: (queryKeys: readonly QueryKey[]) => void;
}

type TabType = "overview" | "resources" | "conditions" | "taints" | "labels";

export function NodeDetailModal({ isOpen, onClose, nodeName }: NodeDetailModalProps) {
  // 延时刷新挂在外层：操作完马上关掉抽屉，稍后那次列表刷新仍然要发生
  const invalidateAfterCommand = useDelayedInvalidate();

  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页等界面状态随关闭丢弃
  if (!isOpen) return null;
  return (
    <NodeDetailDrawer
      key={nodeName}
      onClose={onClose}
      nodeName={nodeName}
      invalidateAfterCommand={invalidateAfterCommand}
    />
  );
}

function NodeDetailDrawer({ onClose, nodeName, invalidateAfterCommand }: NodeDetailDrawerProps) {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const requireAuth = useRequireAuth();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const query = useQuery(nodeDetailQuery(clusterId, nodeName));
  const detail = query.data;
  const scheduling = useMutation(nodeSchedulingMutation(clusterId));

  // 封锁/解封确认框
  const [showCordonConfirm, setShowCordonConfirm] = useState(false);

  // 封锁/解封操作
  const handleCordonClick = () => {
    requireAuth(() => setShowCordonConfirm(true));
  };

  const handleCordonConfirm = () => {
    if (!detail) return;
    scheduling.mutate(
      { nodeName, currentlySchedulable: detail.schedulable },
      {
        onSuccess: () => {
          setShowCordonConfirm(false);
          // 指令由 Agent 异步执行：下发成功时列表与详情已刷新过一次，
          // 晚一点再刷新一次才看得到节点的新状态
          invalidateAfterCommand(clusterKeys.resourceScope("node"));
        },
        onError: (err) => console.error("Cordon/Uncordon failed:", err),
      },
    );
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
      <Drawer isOpen onClose={onClose} title={`Node: ${nodeName}`} size="xl">
        <DetailQueryView query={query} errorFallback={t.common.loadFailed}>
          {(detail) => (
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
          )}
        </DetailQueryView>
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
        loading={scheduling.isPending}
        variant={detail?.schedulable ? "warning" : "info"}
      />
    </>
  );
}
