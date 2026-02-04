"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getStatefulSetDetail, type StatefulSetDetail } from "@/api/workload";
import { getCurrentClusterId } from "@/config/cluster";
import { Server, Box, Settings, Database, Tag } from "lucide-react";

import { OverviewTab, ContainersTab, StrategyTab, StorageTab, LabelsTab } from "./tabs";

interface StatefulSetDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

type TabType = "overview" | "containers" | "strategy" | "storage" | "labels";

export function StatefulSetDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: StatefulSetDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<StatefulSetDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!name || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getStatefulSetDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Name: name,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "加载失败");
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
    { key: "overview", label: "概览", icon: <Server className="w-4 h-4" /> },
    { key: "containers", label: "容器", icon: <Box className="w-4 h-4" /> },
    { key: "strategy", label: "策略", icon: <Settings className="w-4 h-4" /> },
    { key: "storage", label: "存储", icon: <Database className="w-4 h-4" /> },
    { key: "labels", label: "标签", icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`StatefulSet: ${name}`} size="xl">
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
            {activeTab === "containers" && <ContainersTab detail={detail} />}
            {activeTab === "strategy" && <StrategyTab detail={detail} />}
            {activeTab === "storage" && <StorageTab detail={detail} />}
            {activeTab === "labels" && <LabelsTab detail={detail} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}
