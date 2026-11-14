"use client";

import type { Permission } from "@/types/auth";
import { useState } from "react";
import { useQueries, useQuery } from "@tanstack/react-query";
import { FileText, ChevronDown, ChevronRight } from "lucide-react";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { useCurrentCluster } from "@/hooks/useCurrentCluster";
import { configMapListQuery } from "@/queries/cluster";
import { configMapDataQuery } from "@/queries/cluster-content";
import type { useI18n } from "@/i18n/context";

interface ConfigMapsTabProps {
  requireAuth: (action: () => void, permission?: Permission) => boolean;
  namespace: string;
  t: ReturnType<typeof useI18n>["t"];
}

export function ConfigMapsTab({ requireAuth, namespace, t }: ConfigMapsTabProps) {
  const { clusterId } = useCurrentCluster();
  const listQuery = useQuery(configMapListQuery(clusterId, namespace));
  const [expandedNames, setExpandedNames] = useState<Set<string>>(new Set());
  // 展开过的 ConfigMap。内容的查询按这份名单挂着，所以收起再展开不会重新取；
  // 离开本标签页时名单随组件销毁，内容也随之从缓存丢弃
  const [requestedNames, setRequestedNames] = useState<string[]>([]);
  const contentQueries = useQueries({
    queries: requestedNames.map((name) => configMapDataQuery(clusterId, { namespace, name })),
  });
  const contentQueryOf = (name: string) => {
    const position = requestedNames.indexOf(name);
    return position === -1 ? undefined : contentQueries[position];
  };

  const toggleExpand = (name: string) => {
    if (expandedNames.has(name)) {
      setExpandedNames((prev) => {
        const next = new Set(prev);
        next.delete(name);
        return next;
      });
      return;
    }
    requireAuth(() => {
      setExpandedNames((prev) => new Set(prev).add(name));
      const contentQuery = contentQueryOf(name);
      if (contentQuery === undefined) {
        setRequestedNames((prev) => (prev.includes(name) ? prev : [...prev, name]));
      } else if (contentQuery.isError) {
        // 上次没取到的，再展开时重新取一次
        void contentQuery.refetch();
      }
    }, "secret.read");
  };

  const configMaps = listQuery.data;
  if (configMaps === undefined) {
    if (listQuery.isError) {
      return <div className="text-center py-8 text-muted">{t.common.loadFailed}</div>;
    }
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
        const isExpanded = expandedNames.has(cm.name);
        const contentQuery = contentQueryOf(cm.name);
        const isLoadingData = contentQuery?.isFetching ?? false;
        const cmData = contentQuery?.data;

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
