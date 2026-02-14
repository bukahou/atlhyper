"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getResourceQuotaDetail, type ResourceQuotaItem } from "@/api/cluster-resources";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { Server } from "lucide-react";

interface ResourceQuotaDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
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
  const [detail, setDetail] = useState<ResourceQuotaItem | null>(null);

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
          <div className="flex-1 overflow-auto p-6">
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
          </div>
        </div>
      ) : null}
    </Modal>
  );
}
