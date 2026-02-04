"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getNamespaceDetail, getConfigMaps, getSecrets } from "@/api/namespace";
import { getCurrentClusterId } from "@/config/cluster";
import { useRequireAuth } from "@/hooks/useRequireAuth";
import { useI18n } from "@/i18n/context";
import type { NamespaceDetail, ConfigMapDTO, SecretDTO } from "@/types/cluster";
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

export function NamespaceDetailModal({
  isOpen,
  onClose,
  namespaceName,
}: NamespaceDetailModalProps) {
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<NamespaceDetail | null>(null);
  const [configMaps, setConfigMaps] = useState<ConfigMapDTO[]>([]);
  const [configMapsLoading, setConfigMapsLoading] = useState(false);
  const [secrets, setSecrets] = useState<SecretDTO[]>([]);
  const [secretsLoading, setSecretsLoading] = useState(false);
  const requireAuth = useRequireAuth();
  const { t } = useI18n();

  const fetchDetail = useCallback(async () => {
    if (!namespaceName) return;
    setLoading(true);
    setError("");
    try {
      const res = await getNamespaceDetail({
        ClusterID: getCurrentClusterId(),
        Namespace: namespaceName,
      });
      setDetail(res.data.data);
    } catch (err) {
      setError(err instanceof Error ? err.message : t.namespace.loadFailed);
    } finally {
      setLoading(false);
    }
  }, [namespaceName, t.namespace.loadFailed]);

  const fetchConfigMaps = useCallback(async () => {
    if (!namespaceName) return;
    setConfigMapsLoading(true);
    try {
      const res = await getConfigMaps({
        ClusterID: getCurrentClusterId(),
        Namespace: namespaceName,
      });
      setConfigMaps(res.data.data || []);
    } catch (err) {
      console.error("Failed to fetch configmaps:", err);
      setConfigMaps([]);
    } finally {
      setConfigMapsLoading(false);
    }
  }, [namespaceName]);

  const fetchSecrets = useCallback(async () => {
    if (!namespaceName) return;
    setSecretsLoading(true);
    try {
      const res = await getSecrets({
        ClusterID: getCurrentClusterId(),
        Namespace: namespaceName,
      });
      setSecrets(res.data.data || []);
    } catch (err) {
      console.error("Failed to fetch secrets:", err);
      setSecrets([]);
    } finally {
      setSecretsLoading(false);
    }
  }, [namespaceName]);

  useEffect(() => {
    if (isOpen) {
      fetchDetail();
      setActiveTab("overview");
      setConfigMaps([]);
      setSecrets([]);
    }
  }, [isOpen, fetchDetail]);

  // 当切换到 ConfigMap tab 时加载数据
  useEffect(() => {
    if (activeTab === "configmaps" && configMaps.length === 0 && !configMapsLoading) {
      fetchConfigMaps();
    }
  }, [activeTab, configMaps.length, configMapsLoading, fetchConfigMaps]);

  // 当切换到 Secrets tab 时加载数据（需要 Operator 权限）
  useEffect(() => {
    if (activeTab === "secrets" && secrets.length === 0 && !secretsLoading) {
      requireAuth(() => {
        fetchSecrets();
      });
    }
  }, [activeTab, secrets.length, secretsLoading, fetchSecrets, requireAuth]);

  const tabs: { key: TabType; label: string; icon: React.ReactNode }[] = [
    { key: "overview", label: t.namespace.overview, icon: <FolderTree className="w-4 h-4" /> },
    { key: "quotas", label: t.namespace.quotas, icon: <Shield className="w-4 h-4" /> },
    { key: "configmaps", label: t.namespace.configMaps, icon: <FileText className="w-4 h-4" /> },
    { key: "secrets", label: t.namespace.secrets, icon: <Lock className="w-4 h-4" /> },
    { key: "labels", label: t.namespace.labels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`Namespace: ${namespaceName}`} size="xl">
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
            {activeTab === "overview" && <OverviewTab detail={detail} t={t} />}
            {activeTab === "quotas" && (
              <QuotasTab quotas={detail.quotas || []} limitRanges={detail.limitRanges || []} t={t} />
            )}
            {activeTab === "configmaps" && (
              <ConfigMapsTab configMaps={configMaps} loading={configMapsLoading} requireAuth={requireAuth} namespace={namespaceName} t={t} />
            )}
            {activeTab === "secrets" && (
              <SecretsTab secrets={secrets} loading={secretsLoading} requireAuth={requireAuth} namespace={namespaceName} t={t} />
            )}
            {activeTab === "labels" && (
              <LabelsTab labels={detail.labels || {}} annotations={detail.annotations || {}} t={t} />
            )}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}
