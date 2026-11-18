"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { statefulSetDetailQuery } from "@/queries/cluster-workload";
import { Server, Box, Settings, Database, Tag } from "lucide-react";
import { useI18n } from "@/i18n/context";

import { OverviewTab, ContainersTab, StrategyTab, StorageTab, LabelsTab } from "./tabs";

interface StatefulSetDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

type TabType = "overview" | "containers" | "strategy" | "storage" | "labels";

export function StatefulSetDetailModal({ isOpen, onClose, namespace, name }: StatefulSetDetailModalProps) {
  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页等界面状态随关闭丢弃
  if (!isOpen) return null;
  return (
    <StatefulSetDetailDrawer
      key={`${namespace}/${name}`}
      onClose={onClose}
      namespace={namespace}
      name={name}
    />
  );
}

function StatefulSetDetailDrawer({ onClose, namespace, name }: Omit<StatefulSetDetailModalProps, "isOpen">) {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(statefulSetDetailQuery(clusterId, { namespace, name }));
  const [activeTab, setActiveTab] = useState<TabType>("overview");

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.statefulset.overview, icon: <Server className="w-4 h-4" /> },
    { key: "containers", label: t.statefulset.containers, icon: <Box className="w-4 h-4" /> },
    { key: "strategy", label: t.statefulset.strategy, icon: <Settings className="w-4 h-4" /> },
    { key: "storage", label: t.statefulset.storage, icon: <Database className="w-4 h-4" /> },
    { key: "labels", label: t.statefulset.labels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen onClose={onClose} title={`StatefulSet: ${name}`} size="xl">
      <DetailQueryView query={query} errorFallback={t.statefulset.loadFailed}>
        {(detail) => (
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
              {activeTab === "containers" && <ContainersTab detail={detail} t={t} />}
              {activeTab === "strategy" && <StrategyTab detail={detail} t={t} />}
              {activeTab === "storage" && <StorageTab detail={detail} t={t} />}
              {activeTab === "labels" && <LabelsTab detail={detail} t={t} />}
            </div>
          </div>
        )}
      </DetailQueryView>
    </Drawer>
  );
}
