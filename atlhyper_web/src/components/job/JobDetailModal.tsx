"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getJobDetail, type JobDetail } from "@/api/cluster-resources";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { Server, Tag, CheckCircle, XCircle, Clock } from "lucide-react";

interface JobDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

type TabType = "overview" | "labels";

export function JobDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: JobDetailModalProps) {
  const { t } = useI18n();
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
        ClusterID: getCurrentClusterId(),
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
    { key: "labels", label: t.job.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Job: ${name}`} size="xl">
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
            {activeTab === "labels" && <LabelsTab detail={detail} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

// 概览 Tab
function OverviewTab({ detail }: { detail: JobDetail }) {
  const { t } = useI18n();

  const getStatusBadgeType = (status: string): "success" | "warning" | "error" => {
    switch (status) {
      case "Complete": return "success";
      case "Running": return "warning";
      case "Failed": return "error";
      default: return "warning";
    }
  };

  const getStatusLabel = (status: string): string => {
    switch (status) {
      case "Complete": return t.job.statusComplete;
      case "Running": return t.job.statusRunning;
      case "Failed": return t.job.statusFailed;
      default: return status;
    }
  };

  return (
    <div className="space-y-6">
      {/* 状态 Badge */}
      <div className="flex items-center gap-3 p-4 bg-[var(--background)] rounded-lg">
        <Clock className="w-5 h-5 text-primary" />
        <StatusBadge status={getStatusLabel(detail.status)} type={getStatusBadgeType(detail.status)} />
      </div>

      {/* Pod 计数 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Pod {t.common.status}</h3>
        <div className="grid grid-cols-3 gap-3">
          {[
            { label: t.job.active, value: detail.active, color: "text-blue-500", icon: Clock },
            { label: t.job.succeeded, value: detail.succeeded, color: "text-green-500", icon: CheckCircle },
            { label: t.job.failed, value: detail.failed, color: "text-red-500", icon: XCircle },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3 text-center">
              <div className={`text-2xl font-bold ${item.color}`}>{item.value}</div>
              <div className="text-xs text-muted mt-1">{item.label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 时间信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.job.detailTimeInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          {[
            { label: t.job.detailStartTime, value: detail.startTime ? new Date(detail.startTime).toLocaleString() : "-" },
            { label: t.job.detailFinishTime, value: detail.finishTime ? new Date(detail.finishTime).toLocaleString() : "-" },
            { label: t.job.duration, value: detail.duration || "-" },
            { label: "Age", value: detail.age || "-" },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.job.detailBasicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: t.common.name, value: detail.name },
            { label: t.common.namespace, value: detail.namespace },
            { label: "UID", value: detail.uid },
            ...(detail.ownerKind ? [{ label: t.job.detailOwner, value: `${detail.ownerKind}/${detail.ownerName}` }] : []),
            { label: t.job.detailCreatedAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium truncate" title={item.value}>{item.value}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

// 标签 Tab
function LabelsTab({ detail }: { detail: JobDetail }) {
  const { t } = useI18n();
  const labels = Object.entries(detail.labels || {});

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Labels ({labels.length})</h3>
        {labels.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.job.detailNoLabels}</div>
        ) : (
          <div className="space-y-2">
            {labels.map(([key, value]) => (
              <div key={key} className="bg-[var(--background)] rounded-lg p-3 flex items-start gap-2">
                <span className="text-sm font-mono text-primary break-all">{key}</span>
                <span className="text-muted">=</span>
                <span className="text-sm font-mono text-default break-all">{value || '""'}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
