"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { jobDetailQuery } from "@/queries/cluster-workload";
import { useI18n } from "@/i18n/context";
import { Server, Box, Tag } from "lucide-react";
import { OverviewTab, ContainersTab, LabelsTab } from "./JobDetailTabs";

interface JobDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

type TabType = "overview" | "containers" | "labels";

export function JobDetailModal({ isOpen, onClose, namespace, name }: JobDetailModalProps) {
  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页等界面状态随关闭丢弃
  if (!isOpen) return null;
  return <JobDetailDrawer key={`${namespace}/${name}`} onClose={onClose} namespace={namespace} name={name} />;
}

function JobDetailDrawer({ onClose, namespace, name }: Omit<JobDetailModalProps, "isOpen">) {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(jobDetailQuery(clusterId, { namespace, name }));
  const [activeTab, setActiveTab] = useState<TabType>("overview");

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.job.detailOverview, icon: <Server className="w-4 h-4" /> },
    { key: "containers", label: t.job.detailContainers, icon: <Box className="w-4 h-4" /> },
    { key: "labels", label: t.job.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen onClose={onClose} title={`Job: ${name}`} size="xl">
      <DetailQueryView query={query} errorFallback={t.common.loadFailed}>
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
              {activeTab === "overview" && <OverviewTab detail={detail} />}
              {activeTab === "containers" && <ContainersTab detail={detail} />}
              {activeTab === "labels" && <LabelsTab detail={detail} />}
            </div>
          </div>
        )}
      </DetailQueryView>
    </Drawer>
  );
}
