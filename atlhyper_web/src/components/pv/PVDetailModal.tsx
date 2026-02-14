"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { StatusBadge } from "@/components/common";
import { getPVDetail, type PVDetail } from "@/api/cluster-resources";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { Server, Tag } from "lucide-react";

interface PVDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  name: string;
}

type TabType = "overview" | "labels";

export function PVDetailModal({ isOpen, onClose, name }: PVDetailModalProps) {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<PVDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!name) return;
    setLoading(true);
    setError("");
    try {
      const res = await getPVDetail({
        ClusterID: getCurrentClusterId(),
        Name: name,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.common.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [name]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.storagePage.detailOverview, icon: <Server className="w-4 h-4" /> },
    { key: "labels", label: t.storagePage.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`PV: ${name}`} size="xl">
      {loading ? (
        <div className="py-12"><LoadingSpinner /></div>
      ) : error ? (
        <div className="p-6 text-center text-red-500">{error}</div>
      ) : detail ? (
        <div className="flex flex-col h-[70vh]">
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
          <div className="flex-1 overflow-auto p-6">
            {activeTab === "overview" && <OverviewTab detail={detail} />}
            {activeTab === "labels" && <LabelsTab detail={detail} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

function OverviewTab({ detail }: { detail: PVDetail }) {
  const { t } = useI18n();

  const phaseType: Record<string, "success" | "info" | "warning" | "error"> = {
    Bound: "success",
    Available: "info",
    Released: "warning",
    Failed: "error",
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3 p-4 bg-[var(--background)] rounded-lg">
        <Server className="w-5 h-5 text-primary" />
        <StatusBadge status={detail.phase} type={phaseType[detail.phase] || "info"} />
      </div>

      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.storagePage.detailStorageInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: t.storagePage.capacity, value: detail.capacity || "-" },
            { label: t.storagePage.storageClass, value: detail.storageClass || "-" },
            { label: t.storagePage.reclaimPolicy, value: detail.reclaimPolicy || "-" },
            { label: t.storagePage.accessModes, value: detail.accessModes?.join(", ") || "-" },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.storagePage.detailBasicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: t.common.name, value: detail.name },
            { label: "UID", value: detail.uid },
            { label: t.storagePage.detailCreatedAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
            { label: "Age", value: detail.age || "-" },
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

function LabelsTab({ detail }: { detail: PVDetail }) {
  const { t } = useI18n();
  const labels = Object.entries(detail.labels || {});

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Labels ({labels.length})</h3>
        {labels.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.storagePage.detailNoLabels}</div>
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
