"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getDaemonSetDetail } from "@/datasource/cluster";
import type { DaemonSetDetail } from "@/api/workload";
import { useClusterStore } from "@/store/clusterStore";
import { OverviewTab, ContainersTab, StrategyTab, LabelsTab } from "./DaemonSetDetailTabs";
import { Server, Box, Settings, Tag } from "lucide-react";
import { useI18n } from "@/i18n/context";

interface DaemonSetDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

type TabType = "overview" | "containers" | "strategy" | "labels";

export function DaemonSetDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: DaemonSetDetailModalProps) {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<DaemonSetDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!name || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getDaemonSetDetail({
        ClusterID: currentClusterId,
        Namespace: namespace,
        Name: name,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.daemonset.loadFailed);
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
    { key: "overview", label: t.daemonset.overview, icon: <Server className="w-4 h-4" /> },
    { key: "containers", label: t.daemonset.containers, icon: <Box className="w-4 h-4" /> },
    { key: "strategy", label: t.daemonset.strategy, icon: <Settings className="w-4 h-4" /> },
    { key: "labels", label: t.daemonset.labels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen={isOpen} onClose={onClose} title={`DaemonSet: ${name}`} size="xl">
      {loading ? (
        <div className="py-12">
          <LoadingSpinner />
        </div>
      ) : error ? (
        <div className="p-6 text-center text-red-500">{error}</div>
      ) : detail ? (
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
            {activeTab === "labels" && <LabelsTab detail={detail} t={t} />}
          </div>
        </div>
      ) : null}
    </Drawer>
  );
}
