"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { podDetailQuery } from "@/queries/cluster";
import { useI18n } from "@/i18n/context";
import { OverviewTab, ContainersTab, VolumesTab, NetworkTab, SchedulingTab } from "./PodDetailTabs";
import { Box, Container, HardDrive, Network, Settings, History } from "lucide-react";
import { ResourceEventsTab } from "@/components/common/ResourceEventsTab";

interface PodDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  podName: string;
  onViewLogs: (containerName: string) => void;
}

type TabType = "overview" | "containers" | "volumes" | "network" | "scheduling" | "events";

export function PodDetailModal({ isOpen, onClose, namespace, podName, onViewLogs }: PodDetailModalProps) {
  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页等界面状态随关闭丢弃
  if (!isOpen) return null;
  return (
    <PodDetailDrawer
      key={`${namespace}/${podName}`}
      onClose={onClose}
      namespace={namespace}
      podName={podName}
      onViewLogs={onViewLogs}
    />
  );
}

function PodDetailDrawer({ onClose, namespace, podName, onViewLogs }: Omit<PodDetailModalProps, "isOpen">) {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const { clusterId } = useCurrentCluster();
  const query = useQuery(podDetailQuery(clusterId, { namespace, name: podName }));

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.pod.overview, icon: <Box className="w-4 h-4" /> },
    { key: "containers", label: t.pod.containers, icon: <Container className="w-4 h-4" /> },
    { key: "volumes", label: t.pod.volumeMounts, icon: <HardDrive className="w-4 h-4" /> },
    { key: "network", label: t.pod.network, icon: <Network className="w-4 h-4" /> },
    { key: "scheduling", label: t.pod.scheduling, icon: <Settings className="w-4 h-4" /> },
    { key: "events", label: t.nav.event, icon: <History className="w-4 h-4" /> },
  ];

  return (
    <Drawer
      isOpen
      onClose={onClose}
      title={`Pod: ${podName}`}
      size="xl"
    >
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
              {activeTab === "overview" && <OverviewTab detail={detail} t={t} />}
              {activeTab === "containers" && (
                <ContainersTab containers={detail.containers} onViewLogs={onViewLogs} t={t} />
              )}
              {activeTab === "volumes" && <VolumesTab volumes={detail.volumes || []} t={t} />}
              {activeTab === "network" && <NetworkTab detail={detail} t={t} />}
              {activeTab === "scheduling" && <SchedulingTab detail={detail} t={t} />}
              {activeTab === "events" && (
                <ResourceEventsTab clusterId={clusterId} kind="Pod" namespace={detail.namespace} name={detail.name} />
              )}
            </div>
          </div>
        )}
      </DetailQueryView>
    </Drawer>
  );
}
