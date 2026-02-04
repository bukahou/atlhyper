"use client";

import { StatusBadge } from "@/components/common";
import type { ResourceQuotaDTO, LimitRangeDTO } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";

interface QuotasTabProps {
  quotas: ResourceQuotaDTO[];
  limitRanges: LimitRangeDTO[];
  t: ReturnType<typeof useI18n>["t"];
}

export function QuotasTab({ quotas, limitRanges, t }: QuotasTabProps) {
  if (quotas.length === 0 && limitRanges.length === 0) {
    return <div className="text-center py-8 text-muted">{t.namespace.noQuotas}</div>;
  }

  return (
    <div className="space-y-6">
      {/* Resource Quotas */}
      {quotas.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Resource Quotas ({quotas.length})</h3>
          <div className="space-y-3">
            {quotas.map((quota, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-4">
                <h4 className="font-medium text-default mb-3">{quota.name}</h4>
                {quota.scopes && quota.scopes.length > 0 && (
                  <div className="mb-3">
                    <span className="text-xs text-muted">Scopes: </span>
                    {quota.scopes.map((scope, j) => (
                      <StatusBadge key={j} status={scope} type="info" />
                    ))}
                  </div>
                )}
                {quota.hard && Object.keys(quota.hard).length > 0 && (
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                    {Object.entries(quota.hard).map(([key, value]) => (
                      <div key={key} className="bg-card rounded p-2">
                        <div className="text-xs text-muted">{key}</div>
                        <div className="text-sm font-mono">
                          <span className="text-default">{quota.used?.[key] || "0"}</span>
                          <span className="text-muted"> / {value}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Limit Ranges */}
      {limitRanges.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Limit Ranges ({limitRanges.length})</h3>
          <div className="space-y-3">
            {limitRanges.map((lr, i) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-4">
                <h4 className="font-medium text-default mb-3">{lr.name}</h4>
                <div className="space-y-2">
                  {lr.items.map((item, j) => (
                    <div key={j} className="bg-card rounded p-3">
                      <StatusBadge status={item.type} type="info" />
                      <div className="grid grid-cols-2 md:grid-cols-4 gap-2 mt-2 text-xs">
                        {item.default && Object.keys(item.default).length > 0 && (
                          <div>
                            <span className="text-muted">Default: </span>
                            {Object.entries(item.default).map(([k, v]) => (
                              <span key={k} className="font-mono">{k}={v} </span>
                            ))}
                          </div>
                        )}
                        {item.defaultRequest && Object.keys(item.defaultRequest).length > 0 && (
                          <div>
                            <span className="text-muted">Request: </span>
                            {Object.entries(item.defaultRequest).map(([k, v]) => (
                              <span key={k} className="font-mono">{k}={v} </span>
                            ))}
                          </div>
                        )}
                        {item.max && Object.keys(item.max).length > 0 && (
                          <div>
                            <span className="text-muted">Max: </span>
                            {Object.entries(item.max).map(([k, v]) => (
                              <span key={k} className="font-mono">{k}={v} </span>
                            ))}
                          </div>
                        )}
                        {item.min && Object.keys(item.min).length > 0 && (
                          <div>
                            <span className="text-muted">Min: </span>
                            {Object.entries(item.min).map(([k, v]) => (
                              <span key={k} className="font-mono">{k}={v} </span>
                            ))}
                          </div>
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
