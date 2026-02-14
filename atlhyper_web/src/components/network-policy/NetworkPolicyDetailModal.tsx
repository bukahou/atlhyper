"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getNetworkPolicyDetail, type NetworkPolicyItem } from "@/api/cluster-resources";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { Server } from "lucide-react";

interface NetworkPolicyDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

export function NetworkPolicyDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: NetworkPolicyDetailModalProps) {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<NetworkPolicyItem | null>(null);

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
    }
  }, [isOpen, fetchDetail]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title={`NetworkPolicy: ${name}`} size="xl">
      {loading ? (
        <div className="py-12"><LoadingSpinner /></div>
      ) : error ? (
        <div className="p-6 text-center text-red-500">{error}</div>
      ) : detail ? (
        <div className="flex flex-col h-[70vh]">
          <div className="flex-1 overflow-auto p-6">
            <div className="space-y-6">
              {/* 策略类型 */}
              <div className="flex items-center gap-3 p-4 bg-[var(--background)] rounded-lg">
                <Server className="w-5 h-5 text-primary" />
                <span className="text-sm font-medium text-default">
                  {detail.policyTypes?.join(", ") || "-"}
                </span>
              </div>

              {/* 规则统计 */}
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

              {/* 基本信息 */}
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
          </div>
        </div>
      ) : null}
    </Modal>
  );
}
