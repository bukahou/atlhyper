"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { ConfirmDialog } from "@/components/common";
import { getNodeDetail, cordonNode, uncordonNode } from "@/datasource/cluster";
import { useClusterStore } from "@/store/clusterStore";
import { useI18n } from "@/i18n/context";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import type { NodeDetail } from "@/types/cluster";
import { OverviewTab, ResourcesTab, ConditionsTab, TaintsTab, LabelsTab } from "./NodeDetailTabs";
import { Server, Cpu, CheckCircle, AlertTriangle, Tag, Shield, ShieldOff } from "lucide-react";

interface NodeDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  nodeName: string;
  onNodeChanged?: () => void;
}

type TabType = "overview" | "resources" | "conditions" | "taints" | "labels";

export function NodeDetailModal({ isOpen, onClose, nodeName, onNodeChanged }: NodeDetailModalProps) {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
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
        ClusterID: currentClusterId,
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
        await cordonNode({ ClusterID: currentClusterId, Node: nodeName });
      } else {
        await uncordonNode({ ClusterID: currentClusterId, Node: nodeName });
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
