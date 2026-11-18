"use client";

import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Drawer } from "@/components/common/Drawer";
import { DetailQueryView } from "@/components/common/DetailQueryView";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { ingressDetailQuery } from "@/queries/cluster";
import { useI18n } from "@/i18n/context";
import { Globe, Route, Lock, Tag } from "lucide-react";
import { OverviewTab, RulesTab, TLSTab, AnnotationsTab } from "./IngressDetailTabs";

interface IngressDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  ingressName: string;
}

type TabType = "overview" | "rules" | "tls" | "annotations";

export function IngressDetailModal({ isOpen, onClose, namespace, ingressName }: IngressDetailModalProps) {
  // 关着时整个抽屉不挂载：详情查询只在打开期间存在，标签页等界面状态随关闭丢弃
  if (!isOpen) return null;
  return (
    <IngressDetailDrawer
      key={`${namespace}/${ingressName}`}
      onClose={onClose}
      namespace={namespace}
      ingressName={ingressName}
    />
  );
}

function IngressDetailDrawer({ onClose, namespace, ingressName }: Omit<IngressDetailModalProps, "isOpen">) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const { t } = useI18n();
  const { clusterId } = useCurrentCluster();
  const query = useQuery(ingressDetailQuery(clusterId, { namespace, name: ingressName }));

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.ingress.overview, icon: <Globe className="w-4 h-4" /> },
    { key: "rules", label: t.ingress.routingRules, icon: <Route className="w-4 h-4" /> },
    { key: "tls", label: "TLS", icon: <Lock className="w-4 h-4" /> },
    { key: "annotations", label: t.ingress.annotations, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen onClose={onClose} title={`Ingress: ${ingressName}`} size="xl">
      <DetailQueryView query={query} errorFallback={t.ingress.loadFailed}>
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
              {activeTab === "rules" && <RulesTab rules={detail.spec?.rules || []} defaultBackend={detail.spec?.defaultBackend} t={t} />}
              {activeTab === "tls" && <TLSTab tls={detail.spec?.tls || []} t={t} />}
              {activeTab === "annotations" && <AnnotationsTab annotations={detail.annotations || {}} t={t} />}
            </div>
          </div>
        )}
      </DetailQueryView>
    </Drawer>
  );
}
