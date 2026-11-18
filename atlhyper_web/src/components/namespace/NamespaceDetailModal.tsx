"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { PermissionLock } from "@/components/common/PermissionLock";
import { useI18n } from "@/i18n/context";
import { namespaceDetailQuery } from "@/queries/cluster";
import { FolderTree, FileText, Tag, Shield, Lock } from "lucide-react";

// Tab 组件
import {
  OverviewTab,
  QuotasTab,
  ConfigMapsTab,
  SecretsTab,
  LabelsTab,
} from "./tabs";

interface NamespaceDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespaceName: string;
}

type TabType = "overview" | "quotas" | "configmaps" | "secrets" | "labels";

export function NamespaceDetailModal({ isOpen, onClose, namespaceName }: NamespaceDetailModalProps) {
  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页等界面状态随关闭丢弃
  if (!isOpen) return null;
  return <NamespaceDetailDrawer key={namespaceName} onClose={onClose} namespaceName={namespaceName} />;
}

function NamespaceDetailDrawer({ onClose, namespaceName }: Omit<NamespaceDetailModalProps, "isOpen">) {
  const { clusterId } = useCurrentCluster();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const query = useQuery(namespaceDetailQuery(clusterId, namespaceName));
  const requireAuth = useRequireAuth();
  const { t } = useI18n();

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.namespace.overview, icon: <FolderTree className="w-4 h-4" /> },
    { key: "quotas", label: t.namespace.quotas, icon: <Shield className="w-4 h-4" /> },
    { key: "configmaps", label: t.namespace.configMaps, icon: <FileText className="w-4 h-4" /> },
    { key: "secrets", label: t.namespace.secrets, icon: <Lock className="w-4 h-4" /> },
    { key: "labels", label: t.namespace.labels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen onClose={onClose} title={`Namespace: ${namespaceName}`} size="xl">
      <DetailQueryView query={query} errorFallback={t.namespace.loadFailed}>
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
              {activeTab === "quotas" && (
                <QuotasTab quotas={detail.quotas || []} limitRanges={detail.limitRanges || []} t={t} />
              )}
              {activeTab === "configmaps" && (
                <ConfigMapsTab requireAuth={requireAuth} namespace={namespaceName} t={t} />
              )}
              {activeTab === "secrets" && (
                <PermissionLock permission="secret.read">
                  <SecretsTab requireAuth={requireAuth} namespace={namespaceName} t={t} />
                </PermissionLock>
              )}
              {activeTab === "labels" && (
                <LabelsTab labels={detail.labels || {}} annotations={detail.annotations || {}} t={t} />
              )}
            </div>
          </div>
        )}
      </DetailQueryView>
    </Drawer>
  );
}
