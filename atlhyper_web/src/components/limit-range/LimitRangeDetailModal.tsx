"use client";

import { useState, useEffect, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getLimitRangeDetail, type LimitRangeItem } from "@/api/cluster-resources";
import { getCurrentClusterId } from "@/config/cluster";
import { useI18n } from "@/i18n/context";
import { Server } from "lucide-react";

interface LimitRangeDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  name: string;
}

export function LimitRangeDetailModal({
  isOpen,
  onClose,
  namespace,
  name,
}: LimitRangeDetailModalProps) {
  const { t } = useI18n();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [detail, setDetail] = useState<LimitRangeItem | null>(null);

  const fetchDetail = useCallback(async () => {
    if (!name || !namespace) return;
    setLoading(true);
    setError("");
    try {
      const res = await getLimitRangeDetail({
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
    <Modal isOpen={isOpen} onClose={onClose} title={`LimitRange: ${name}`} size="xl">
      {loading ? (
        <div className="py-12"><LoadingSpinner /></div>
      ) : error ? (
        <div className="p-6 text-center text-red-500">{error}</div>
      ) : detail ? (
        <div className="flex flex-col h-[70vh]">
          <div className="flex-1 overflow-auto p-6">
            <div className="space-y-6">
              {/* 限制项 */}
              <div>
                <h3 className="text-sm font-semibold text-default mb-3">
                  {t.policyPage.detailLimitEntries} ({detail.items.length})
                </h3>
                {detail.items.length === 0 ? (
                  <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">
                    {t.policyPage.detailNoEntries}
                  </div>
                ) : (
                  <div className="space-y-3">
                    {detail.items.map((entry, idx) => (
                      <div key={idx} className="bg-[var(--background)] rounded-lg p-4">
                        <div className="flex items-center gap-2 mb-3">
                          <Server className="w-4 h-4 text-primary" />
                          <span className="text-sm font-medium text-default">{entry.type}</span>
                        </div>
                        <div className="grid grid-cols-2 md:grid-cols-3 gap-2 text-xs">
                          {entry.max && Object.keys(entry.max).length > 0 && (
                            <div>
                              <div className="text-muted mb-1">Max</div>
                              {Object.entries(entry.max).map(([k, v]) => (
                                <div key={k} className="font-mono text-default">{k}: {v}</div>
                              ))}
                            </div>
                          )}
                          {entry.min && Object.keys(entry.min).length > 0 && (
                            <div>
                              <div className="text-muted mb-1">Min</div>
                              {Object.entries(entry.min).map(([k, v]) => (
                                <div key={k} className="font-mono text-default">{k}: {v}</div>
                              ))}
                            </div>
                          )}
                          {entry.default && Object.keys(entry.default).length > 0 && (
                            <div>
                              <div className="text-muted mb-1">Default</div>
                              {Object.entries(entry.default).map(([k, v]) => (
                                <div key={k} className="font-mono text-default">{k}: {v}</div>
                              ))}
                            </div>
                          )}
                          {entry.defaultRequest && Object.keys(entry.defaultRequest).length > 0 && (
                            <div>
                              <div className="text-muted mb-1">Default Request</div>
                              {Object.entries(entry.defaultRequest).map(([k, v]) => (
                                <div key={k} className="font-mono text-default">{k}: {v}</div>
                              ))}
                            </div>
                          )}
                          {entry.maxLimitRequestRatio && Object.keys(entry.maxLimitRequestRatio).length > 0 && (
                            <div>
                              <div className="text-muted mb-1">Max Ratio</div>
                              {Object.entries(entry.maxLimitRequestRatio).map(([k, v]) => (
                                <div key={k} className="font-mono text-default">{k}: {v}</div>
                              ))}
                            </div>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>

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
