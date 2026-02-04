"use client";

import { useState } from "react";
import { FileText, ChevronDown, ChevronRight } from "lucide-react";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getConfigMapData } from "@/api/namespace";
import { getCurrentClusterId } from "@/config/cluster";
import type { ConfigMapDTO } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";

interface ConfigMapsTabProps {
  configMaps: ConfigMapDTO[];
  loading: boolean;
  requireAuth: (action: () => void) => boolean;
  namespace: string;
  t: ReturnType<typeof useI18n>["t"];
}

export function ConfigMapsTab({
  configMaps,
  loading,
  requireAuth,
  namespace,
  t,
}: ConfigMapsTabProps) {
  const [expandedCMs, setExpandedCMs] = useState<Set<string>>(new Set());
  const [cmDataMap, setCmDataMap] = useState<Record<string, Record<string, string>>>({});
  const [loadingCMs, setLoadingCMs] = useState<Set<string>>(new Set());

  const toggleExpand = async (name: string) => {
    const isExpanded = expandedCMs.has(name);
    if (isExpanded) {
      setExpandedCMs((prev) => {
        const next = new Set(prev);
        next.delete(name);
        return next;
      });
    } else {
      requireAuth(async () => {
        setExpandedCMs((prev) => {
          const next = new Set(prev);
          next.add(name);
          return next;
        });

        if (cmDataMap[name]) return;

        setLoadingCMs((prev) => new Set(prev).add(name));
        try {
          const data = await getConfigMapData({
            ClusterID: getCurrentClusterId(),
            Namespace: namespace,
            Name: name,
          });
          setCmDataMap((prev) => ({ ...prev, [name]: data }));
        } catch (err) {
          console.error("Failed to fetch ConfigMap data:", err);
        } finally {
          setLoadingCMs((prev) => {
            const next = new Set(prev);
            next.delete(name);
            return next;
          });
        }
      });
    }
  };

  if (loading) {
    return (
      <div className="py-8">
        <LoadingSpinner />
      </div>
    );
  }

  if (configMaps.length === 0) {
    return <div className="text-center py-8 text-muted">{t.namespace.noConfigMaps}</div>;
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + " " + sizes[i];
  };

  return (
    <div className="space-y-3">
      {configMaps.map((cm) => {
        const isExpanded = expandedCMs.has(cm.name);
        const isLoadingData = loadingCMs.has(cm.name);
        const cmData = cmDataMap[cm.name];

        return (
          <div key={cm.name} className="bg-[var(--background)] rounded-lg overflow-hidden">
            <button
              onClick={() => toggleExpand(cm.name)}
              className="w-full p-4 flex items-center justify-between hover:bg-[var(--border-color)]/30 transition-colors"
            >
              <div className="flex items-center gap-3">
                <FileText className="w-5 h-5 text-primary" />
                <div className="text-left">
                  <div className="font-medium text-default">{cm.name}</div>
                  <div className="text-xs text-muted">
                    {cm.keys} keys · {formatBytes(cm.totalSizeBytes)}
                    {cm.immutable && <span className="ml-2 text-yellow-500">(Immutable)</span>}
                  </div>
                </div>
              </div>
              {isExpanded ? (
                <ChevronDown className="w-4 h-4 text-muted" />
              ) : (
                <ChevronRight className="w-4 h-4 text-muted" />
              )}
            </button>

            {isExpanded && (
              <div className="px-4 pb-4 border-t border-[var(--border-color)]">
                {isLoadingData ? (
                  <div className="py-4 flex justify-center">
                    <LoadingSpinner />
                  </div>
                ) : cmData && Object.keys(cmData).length > 0 ? (
                  <div className="mt-3 space-y-2">
                    {Object.entries(cmData).map(([key, value]) => (
                      <div key={key} className="bg-card rounded p-3">
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-mono text-sm text-primary">{key}</span>
                          <span className="text-xs text-muted">{value.length} bytes</span>
                        </div>
                        <pre className="text-xs text-muted bg-[var(--background)] p-2 rounded overflow-x-auto max-h-48 whitespace-pre-wrap break-all">
                          {value}
                        </pre>
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="py-4 text-center text-muted text-sm">{t.namespace.noData}</div>
                )}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}
