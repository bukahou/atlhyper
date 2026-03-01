"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getJobDetail } from "@/datasource/cluster";
import type { JobDetail } from "@/api/cluster-resources";
import { useClusterStore } from "@/store/clusterStore";
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

export function JobDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: JobDetailModalProps) {
  const { t } = useI18n();
  const { currentClusterId } = useClusterStore();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<JobDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!name || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getJobDetail({
        ClusterID: currentClusterId,
        Namespace: namespace,
        Name: name,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
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
    { key: "overview", label: t.job.detailOverview, icon: <Server className="w-4 h-4" /> },
    { key: "containers", label: t.job.detailContainers, icon: <Box className="w-4 h-4" /> },
    { key: "labels", label: t.job.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen={isOpen} onClose={onClose} title={`Job: ${name}`} size="xl">
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
            {activeTab === "overview" && <OverviewTab detail={detail} />}
            {activeTab === "containers" && <ContainersTab detail={detail} />}
            {activeTab === "labels" && <LabelsTab detail={detail} />}
          </div>
        </div>
      ) : null}
    </Drawer>
  );
}
