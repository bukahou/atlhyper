"use client";

import { useState, useEffect, useCallback } from "react";
import { Drawer } from "@/components/common/Drawer";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getIngressDetail } from "@/datasource/cluster";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import type { IngressDetail } from "@/types/cluster";
import { Globe, Route, Lock, Tag } from "lucide-react";
import { OverviewTab, RulesTab, TLSTab, AnnotationsTab } from "./IngressDetailTabs";

interface IngressDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  ingressName: string;
}

type TabType = "overview" | "rules" | "tls" | "annotations";

export function IngressDetailModal({
  isOpen,
  onClose,
  namespace,
  ingressName,
}: IngressDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<IngressDetail | null>(null);
  const { t } = useI18n();

  const fetchDetail = useCallback(async () => {
    if (!ingressName || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getIngressDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Name: ingressName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.ingress.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [namespace, ingressName, t.ingress.loadFailed]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
    }
  }, [isOpen, fetchDetail]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.ingress.overview, icon: <Globe className="w-4 h-4" /> },
    { key: "rules", label: t.ingress.routingRules, icon: <Route className="w-4 h-4" /> },
    { key: "tls", label: "TLS", icon: <Lock className="w-4 h-4" /> },
    { key: "annotations", label: t.ingress.annotations, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Drawer isOpen={isOpen} onClose={onClose} title={`Ingress: ${ingressName}`} size="xl">
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
            {activeTab === "overview" && <OverviewTab detail={detail} t={t} />}
            {activeTab === "rules" && <RulesTab rules={detail.spec?.rules || []} defaultBackend={detail.spec?.defaultBackend} t={t} />}
            {activeTab === "tls" && <TLSTab tls={detail.spec?.tls || []} t={t} />}
            {activeTab === "annotations" && <AnnotationsTab annotations={detail.annotations || {}} t={t} />}
          </div>
        </div>
      ) : null}
    </Drawer>
  );
}
