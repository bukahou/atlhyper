"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getCronJobDetail, type CronJobDetail } from "@/api/cluster-resources";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { Server, Tag, Calendar, Clock } from "lucide-react";

interface CronJobDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

type TabType = "overview" | "labels";

export function CronJobDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: CronJobDetailModalProps) {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<CronJobDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!name || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getCronJobDetail({
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
    { key: "overview", label: t.cronjob.detailOverview, icon: <Server className="w-4 h-4" /> },
    { key: "labels", label: t.cronjob.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`CronJob: ${name}`} size="xl">
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
function OverviewTab({ detail }: { detail: CronJobDetail }) {
  const { t } = useI18n();

  return (
    <div className="space-y-6">
      {/* Suspend 状态 + Active Jobs */}
      <div className="flex items-center gap-3 p-4 bg-[var(--background)] rounded-lg">
        <Calendar className="w-5 h-5 text-primary" />
        <StatusBadge
          status={detail.suspend ? t.cronjob.suspended : t.common.enabled}
          type={detail.suspend ? "error" : "success"}
        />
        {detail.activeJobs > 0 && (
          <StatusBadge status={`${t.cronjob.activeJobs}: ${detail.activeJobs}`} type="info" />
        )}
      </div>

      {/* 调度信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.cronjob.detailScheduleInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.cronjob.schedule}</div>
            <div className="text-sm text-default font-mono font-medium">{detail.schedule}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.cronjob.lastSchedule}</div>
            <div className="text-sm text-default font-medium">
              {detail.lastScheduleTime ? new Date(detail.lastScheduleTime).toLocaleString() : "-"}
            </div>
            {detail.lastScheduleAgo && (
              <div className="text-xs text-muted mt-0.5">
                <Clock className="w-3 h-3 inline mr-1" />{detail.lastScheduleAgo} ago
              </div>
            )}
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.cronjob.detailLastSuccess}</div>
            <div className="text-sm text-default font-medium">
              {detail.lastSuccessfulTime ? new Date(detail.lastSuccessfulTime).toLocaleString() : "-"}
            </div>
            {detail.lastSuccessAgo && (
              <div className="text-xs text-muted mt-0.5">
                <Clock className="w-3 h-3 inline mr-1" />{detail.lastSuccessAgo} ago
              </div>
            )}
          </div>
        </div>
      </div>

      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.cronjob.detailBasicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: t.common.name, value: detail.name },
            { label: t.common.namespace, value: detail.namespace },
            { label: "UID", value: detail.uid },
            ...(detail.ownerKind ? [{ label: t.cronjob.detailOwner, value: `${detail.ownerKind}/${detail.ownerName}` }] : []),
            { label: "Age", value: detail.age || "-" },
            { label: t.cronjob.detailCreatedAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
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
function LabelsTab({ detail }: { detail: CronJobDetail }) {
  const { t } = useI18n();
  const labels = Object.entries(detail.labels || {});

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Labels ({labels.length})</h3>
        {labels.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.cronjob.detailNoLabels}</div>
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
