"use client";

import { useState } from "react";
import { Lock, ChevronDown, ChevronRight } from "lucide-react";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getSecretData } from "@/api/namespace";
import { getCurrentClusterId } from "@/config/cluster";
import type { SecretDTO } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";

interface SecretsTabProps {
  secrets: SecretDTO[];
  loading: boolean;
  requireAuth: (action: () => void) => boolean;
  namespace: string;
  t: ReturnType<typeof useI18n>["t"];
}

export function SecretsTab({
  secrets,
  loading,
  requireAuth,
  namespace,
  t,
}: SecretsTabProps) {
  const [expandedSecrets, setExpandedSecrets] = useState<Set<string>>(new Set());
  const [secretDataMap, setSecretDataMap] = useState<Record<string, Record<string, string>>>({});
  const [loadingSecrets, setLoadingSecrets] = useState<Set<string>>(new Set());

  const toggleExpand = async (name: string) => {
    const isExpanded = expandedSecrets.has(name);
    if (isExpanded) {
      setExpandedSecrets((prev) => {
        const next = new Set(prev);
        next.delete(name);
        return next;
      });
    } else {
      requireAuth(async () => {
        setExpandedSecrets((prev) => {
          const next = new Set(prev);
          next.add(name);
          return next;
        });

        if (secretDataMap[name]) return;

        setLoadingSecrets((prev) => new Set(prev).add(name));
        try {
          const data = await getSecretData({
            ClusterID: getCurrentClusterId(),
            Namespace: namespace,
            Name: name,
          });
          setSecretDataMap((prev) => ({ ...prev, [name]: data }));
        } catch (err) {
          console.error("Failed to fetch Secret data:", err);
        } finally {
          setLoadingSecrets((prev) => {
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

  if (secrets.length === 0) {
    return <div className="text-center py-8 text-muted">{t.namespace.noSecrets}</div>;
  }

  return (
    <div className="space-y-3">
      {secrets.map((secret) => {
        const isExpanded = expandedSecrets.has(secret.name);
        const isLoadingData = loadingSecrets.has(secret.name);
        const secretData = secretDataMap[secret.name];

        return (
          <div key={secret.name} className="bg-[var(--background)] rounded-lg overflow-hidden">
            <button
              onClick={() => toggleExpand(secret.name)}
              className="w-full p-4 flex items-center justify-between hover:bg-[var(--border-color)]/30 transition-colors"
            >
              <div className="flex items-center gap-3">
                <Lock className="w-5 h-5 text-yellow-500" />
                <div className="text-left">
                  <div className="font-medium text-default">{secret.name}</div>
                  <div className="text-xs text-muted">
                    {secret.type} · {secret.dataKeys?.length || 0} keys
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
                ) : secretData && Object.keys(secretData).length > 0 ? (
                  <div className="mt-3 space-y-2">
                    {Object.entries(secretData).map(([key, value]) => (
                      <div key={key} className="bg-card rounded p-3">
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-mono text-sm text-yellow-500">{key}</span>
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
