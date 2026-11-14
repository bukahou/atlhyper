"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import type { ResourceQuotaDetail } from "@/api/cluster-resources";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { resourceQuotaDetailQuery } from "@/queries/cluster-policy";
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

export function ResourceQuotaDetailModal({ isOpen, onClose, namespace, name }: ResourceQuotaDetailModalProps) {
  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页等界面状态随关闭丢弃
  if (!isOpen) return null;
  return (
    <ResourceQuotaDetailDrawer
      key={`${namespace}/${name}`}
      onClose={onClose}
      namespace={namespace}
      name={name}
    />
  );
}

function ResourceQuotaDetailDrawer({ onClose, namespace, name }: Omit<ResourceQuotaDetailModalProps, "isOpen">) {
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(resourceQuotaDetailQuery(clusterId, { namespace, name }));
  const [activeTab, setActiveTab] = useState<TabType>("overview");

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.policyPage.detailOverview, icon: <Server className="w-4 h-4" /> },
    { key: "labels", label: t.policyPage.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen onClose={onClose} title={`ResourceQuota: ${name}`} size="xl">
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
