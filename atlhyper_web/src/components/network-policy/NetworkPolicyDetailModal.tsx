"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getNetworkPolicyDetail, type NetworkPolicyDetail, type NetworkPolicyRule } from "@/api/cluster-resources";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { Server, Shield, Tag } from "lucide-react";

interface NetworkPolicyDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

type TabType = "overview" | "rules" | "labels";

export function NetworkPolicyDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: NetworkPolicyDetailModalProps) {
  const { t } = useI18n();
  const [activeTab, setActiveTab] = useState<TabType>("overview");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<NetworkPolicyDetail | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!name || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getNetworkPolicyDetail({
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
    { key: "overview", label: t.policyPage.detailOverview, icon: <Server className="w-4 h-4" /> },
    { key: "rules", label: t.policyPage.detailRules, icon: <Shield className="w-4 h-4" /> },
    { key: "labels", label: t.policyPage.detailLabels, icon: <Tag className="w-4 h-4" /> },
  ];

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`NetworkPolicy: ${name}`} size="xl">
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
            {activeTab === "rules" && <RulesTab detail={detail} />}
            {activeTab === "labels" && <LabelsTab detail={detail} />}
          </div>
        </div>
      ) : null}
    </Modal>
  );
}

function OverviewTab({ detail }: { detail: NetworkPolicyDetail }) {
  const { t } = useI18n();

  return (
    <div className="space-y-6">
      {/* Policy Types */}
      <div className="flex items-center gap-3 p-4 bg-[var(--background)] rounded-lg">
        <Server className="w-5 h-5 text-primary" />
        <span className="text-sm font-medium text-default">
          {detail.policyTypes?.join(", ") || "-"}
        </span>
      </div>

      {/* Rule Stats */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.policyPage.detailRuleStats}</h3>
        <div className="grid grid-cols-2 gap-3">
          <div className="bg-[var(--background)] rounded-lg p-3 text-center">
            <div className="text-2xl font-bold text-purple-500">{detail.ingressRuleCount}</div>
            <div className="text-xs text-muted mt-1">{t.policyPage.ingressRules}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3 text-center">
            <div className="text-2xl font-bold text-green-500">{detail.egressRuleCount}</div>
            <div className="text-xs text-muted mt-1">{t.policyPage.egressRules}</div>
          </div>
        </div>
      </div>

      {/* Basic Info */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.policyPage.detailBasicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: t.common.name, value: detail.name },
            { label: t.common.namespace, value: detail.namespace },
            { label: t.policyPage.detailPodSelector, value: detail.podSelector || "-" },
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

function RuleCard({ rule, index }: { rule: NetworkPolicyRule; index: number }) {
  const { t } = useI18n();
  const peers = rule.peers || [];
  const ports = rule.ports || [];

  return (
    <div className="bg-[var(--background)] rounded-lg p-4 space-y-3">
      <div className="text-sm font-medium text-default">Rule #{index + 1}</div>

      {/* Peers */}
      <div>
        <div className="text-xs font-semibold text-muted mb-2">{t.policyPage.detailPeers}</div>
        {peers.length === 0 ? (
          <div className="text-xs text-muted italic">-</div>
        ) : (
          <div className="space-y-1.5">
            {peers.map((peer, i) => (
              <div key={i} className="flex items-start gap-2 text-xs bg-[var(--card-bg)] rounded p-2">
                <span className="shrink-0 px-1.5 py-0.5 rounded bg-purple-500/10 text-purple-500 font-medium">
                  {peer.type}
                </span>
                <span className="font-mono text-default break-all">
                  {peer.selector || peer.cidr || "-"}
                  {peer.except && peer.except.length > 0 && (
                    <span className="text-muted"> (except: {peer.except.join(", ")})</span>
                  )}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Ports */}
      <div>
        <div className="text-xs font-semibold text-muted mb-2">{t.policyPage.detailPorts}</div>
        {ports.length === 0 ? (
          <div className="text-xs text-muted italic">-</div>
        ) : (
          <div className="flex flex-wrap gap-2">
            {ports.map((port, i) => (
              <span key={i} className="inline-flex items-center gap-1 px-2 py-1 rounded bg-[var(--card-bg)] text-xs font-mono text-default">
                <span className="text-green-500">{port.protocol}</span>
                <span className="text-muted">/</span>
                <span>{port.port}</span>
                {port.endPort && (
                  <>
                    <span className="text-muted">-</span>
                    <span>{port.endPort}</span>
                  </>
                )}
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function RulesTab({ detail }: { detail: NetworkPolicyDetail }) {
  const { t } = useI18n();
  const ingressRules = detail.ingressRules || [];
  const egressRules = detail.egressRules || [];
  const hasNoRules = ingressRules.length === 0 && egressRules.length === 0;

  if (hasNoRules) {
    return (
      <div className="text-center py-8 text-muted bg-[var(--background)] rounded-lg">
        {t.policyPage.detailNoRules}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Ingress Rules */}
      {ingressRules.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-purple-500" />
            {t.policyPage.ingressRules} ({ingressRules.length})
          </h3>
          <div className="space-y-3">
            {ingressRules.map((rule, i) => (
              <RuleCard key={i} rule={rule} index={i} />
            ))}
          </div>
        </div>
      )}

      {/* Egress Rules */}
      {egressRules.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3 flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-green-500" />
            {t.policyPage.egressRules} ({egressRules.length})
          </h3>
          <div className="space-y-3">
            {egressRules.map((rule, i) => (
              <RuleCard key={i} rule={rule} index={i} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function LabelsTab({ detail }: { detail: NetworkPolicyDetail }) {
  const { t } = useI18n();
  const labels = Object.entries(detail.labels || {});

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
        <h3 className="text-sm font-semibold text-default mb-3">Annotations ({Object.keys(detail.annotations || {}).length})</h3>
        {Object.keys(detail.annotations || {}).length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.policyPage.detailNoAnnotations}</div>
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
