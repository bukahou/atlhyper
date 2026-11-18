"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import { StatusBadge } from "@/components/common";
import type { PVDetail } from "@/api/cluster-resources";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { pvDetailQuery } from "@/queries/cluster-storage";
import { useI18n } from "@/i18n/context";
import { Server, Tag } from "lucide-react";

interface PVDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  name: string;
}

type TabType = "overview" | "labels";

export function PVDetailModal({ isOpen, onClose, name }: PVDetailModalProps) {
  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页等界面状态随关闭丢弃
  if (!isOpen) return null;
  return <PVDetailDrawer key={name} onClose={onClose} name={name} />;
}

function PVDetailDrawer({ onClose, name }: Omit<PVDetailModalProps, "isOpen">) {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(pvDetailQuery(clusterId, name));
  const [activeTab, setActiveTab] = useState<TabType>("overview");

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.storagePage.detailOverview, icon: <Server className="w-4 h-4" /> },
    { key: "labels", label: t.storagePage.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen onClose={onClose} title={`PV: ${name}`} size="xl">
      <DetailQueryView query={query} errorFallback={t.common.loadFailed}>
        {(detail) => (
          <div className="flex flex-col h-full">
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
        )}
      </DetailQueryView>
    </Drawer>
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
            { label: t.storagePage.detailVolumeSourceType, value: detail.volumeSourceType || "-" },
            { label: t.storagePage.detailClaimRef, value: detail.claimRefName ? `${detail.claimRefNamespace}/${detail.claimRefName}` : "-" },
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
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Annotations ({Object.keys(detail.annotations || {}).length})</h3>
        {Object.keys(detail.annotations || {}).length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.storagePage.detailNoAnnotations}</div>
        ) : (
          <div className="space-y-2">
            {Object.entries(detail.annotations || {}).map(([key, value]) => (
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
