"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getResourceQuotaDetail, type ResourceQuotaDetail } from "@/api/cluster-resources";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { Server, Tag } from "lucide-react";

type TabType = "overview" | "labels";

interface ResourceQuotaDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

function OverviewTab({ detail }: { detail: ResourceQuotaDetail }) {
  const { t } = useI18n();

  return (
    <div className="space-y-6">
      {/* 配额使用 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.policyPage.detailQuotaUsage}</h3>
        <div className="space-y-2">
          {Object.keys(detail.hard || {}).map((key) => {
            const hard = detail.hard?.[key] || "-";
            const used = detail.used?.[key] || "0";
            return (
              <div key={key} className="bg-[var(--background)] rounded-lg p-3">
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-mono text-primary">{key}</span>
                  <span className="text-sm text-default font-medium">{used} / {hard}</span>
                </div>
              </div>
            );
          })}
          {Object.keys(detail.hard || {}).length === 0 && (
            <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">
              {t.policyPage.detailNoQuota}
            </div>
          )}
        </div>
      </div>

      {/* Scopes */}
      {detail.scopes && detail.scopes.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.policyPage.detailScopes}</h3>
          <div className="flex flex-wrap gap-2">
            {detail.scopes.map((scope) => (
              <span key={scope} className="px-2 py-1 text-xs bg-primary/10 text-primary rounded">
                {scope}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.policyPage.detailBasicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: t.common.name, value: detail.name },
            { label: t.common.namespace, value: detail.namespace },
            { label: t.policyPage.detailCreatedAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
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

function LabelsTab({ detail }: { detail: ResourceQuotaDetail }) {
  const { t } = useI18n();
  const labels = Object.entries(detail.labels || {});
  const annotations = Object.entries(detail.annotations || {});

  return (
    <div className="space-y-6">
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Labels ({labels.length})</h3>
        {labels.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.policyPage.detailNoLabels}</div>
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
        <h3 className="text-sm font-semibold text-default mb-3">Annotations ({annotations.length})</h3>
        {annotations.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.policyPage.detailNoAnnotations}</div>
        ) : (
          <div className="space-y-2">
            {annotations.map(([key, value]) => (
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

export function ResourceQuotaDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: ResourceQuotaDetailModalProps) {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<ResourceQuotaDetail | null>(null);
  const [activeTab, setActiveTab] = useState<TabType>("overview");

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.policyPage.detailOverview, icon: <Server className="w-4 h-4" /> },
    { key: "labels", label: t.policyPage.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  const fetchDetail = useCallback(async () => {
    if (!name || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getResourceQuotaDetail({
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
      setActiveTab("overview");
      fetchDetail();
    }
  }, [isOpen, fetchDetail]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`ResourceQuota: ${name}`} size="xl">
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
