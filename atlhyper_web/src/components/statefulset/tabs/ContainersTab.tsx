"use client";

import { StatusBadge } from "@/components/common";
import type { StatefulSetDetail } from "@/api/workload";

interface ContainersTabProps {
  detail: StatefulSetDetail;
}

export function ContainersTab({ detail }: ContainersTabProps) {
  const containers = detail.template?.containers || [];

  if (containers.length === 0) {
    return <div className="text-center py-8 text-muted">暂无容器信息</div>;
  }

  return (
    <div className="space-y-4">
      {containers.map((c, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center justify-between mb-3">
            <h4 className="font-medium text-default">{c.name}</h4>
            {c.imagePullPolicy && (
              <StatusBadge status={c.imagePullPolicy} type="info" />
            )}
          </div>

          <div className="space-y-3">
            {/* 镜像 */}
            <div>
              <span className="text-xs text-muted">镜像: </span>
              <span className="text-sm font-mono text-default break-all">{c.image}</span>
            </div>

            {/* 端口 */}
            {c.ports && c.ports.length > 0 && (
              <div>
                <span className="text-xs text-muted block mb-1">端口:</span>
                <div className="flex flex-wrap gap-2">
                  {c.ports.map((p, j) => (
                    <span key={j} className="px-2 py-1 bg-[var(--card-background)] rounded text-xs font-mono">
                      {p.name ? `${p.name}:` : ""}{p.containerPort}/{p.protocol || "TCP"}
                    </span>
                  ))}
                </div>
              </div>
            )}

            {/* 资源 */}
            {(c.requests || c.limits) && (
              <div className="grid grid-cols-2 gap-3">
                {c.requests && Object.keys(c.requests).length > 0 && (
                  <div>
                    <span className="text-xs text-muted block mb-1">Requests:</span>
                    <div className="text-sm font-mono">
                      {Object.entries(c.requests).map(([k, v]) => (
                        <div key={k}>{k}: {v}</div>
                      ))}
                    </div>
                  </div>
                )}
                {c.limits && Object.keys(c.limits).length > 0 && (
                  <div>
                    <span className="text-xs text-muted block mb-1">Limits:</span>
                    <div className="text-sm font-mono">
                      {Object.entries(c.limits).map(([k, v]) => (
                        <div key={k}>{k}: {v}</div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Probes */}
            {(c.livenessProbe || c.readinessProbe || c.startupProbe) && (
              <div className="grid grid-cols-3 gap-2 mt-2">
                {c.livenessProbe && (
                  <div className="p-2 bg-[var(--card-background)] rounded">
                    <div className="text-xs text-green-500 mb-1">Liveness</div>
                    <div className="text-xs text-muted">{c.livenessProbe.type}</div>
                  </div>
                )}
                {c.readinessProbe && (
                  <div className="p-2 bg-[var(--card-background)] rounded">
                    <div className="text-xs text-blue-500 mb-1">Readiness</div>
                    <div className="text-xs text-muted">{c.readinessProbe.type}</div>
                  </div>
                )}
                {c.startupProbe && (
                  <div className="p-2 bg-[var(--card-background)] rounded">
                    <div className="text-xs text-purple-500 mb-1">Startup</div>
                    <div className="text-xs text-muted">{c.startupProbe.type}</div>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}
