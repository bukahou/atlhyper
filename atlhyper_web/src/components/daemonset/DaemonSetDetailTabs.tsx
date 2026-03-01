"use client";

import { StatusBadge } from "@/components/common";
import type { DaemonSetDetail } from "@/api/workload";
import {
  Server,
  Activity,
  CheckCircle,
  AlertTriangle,
  XCircle,
} from "lucide-react";
import type { useI18n } from "@/i18n/context";

type Translations = ReturnType<typeof useI18n>["t"];

interface TabProps {
  detail: DaemonSetDetail;
  t: Translations;
}

// 概览 Tab
export function OverviewTab({ detail, t }: TabProps) {
  const getRolloutBadgeType = (phase: string): "success" | "warning" | "error" | "info" => {
    switch (phase) {
      case "Complete": return "success";
      case "Progressing": return "info";
      case "Degraded": return "error";
      default: return "warning";
    }
  };

  return (
    <div className="space-y-6">
      {/* Rollout Status */}
      {detail.rollout && (
        <div className="flex items-center gap-3 p-4 bg-[var(--background)] rounded-lg">
          <Activity className="w-5 h-5 text-primary" />
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <StatusBadge status={detail.rollout.phase} type={getRolloutBadgeType(detail.rollout.phase)} />
              {detail.rollout.badges?.map((badge, i) => (
                <StatusBadge key={i} status={badge} type={badge === "Misscheduled" || badge === "Unavailable" ? "error" : "info"} />
              ))}
            </div>
            {detail.rollout.message && (
              <p className="text-sm text-muted mt-1">{detail.rollout.message}</p>
            )}
          </div>
        </div>
      )}

      {/* 节点调度状态 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.daemonset.nodeScheduleStatus}</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          {[
            { label: t.daemonset.desired, value: detail.desired, color: "text-default", icon: Server },
            { label: t.daemonset.ready, value: detail.ready, color: "text-green-500", icon: CheckCircle },
            { label: t.daemonset.available, value: detail.available, color: "text-blue-500", icon: CheckCircle },
            { label: t.daemonset.current, value: detail.current, color: "text-purple-500", icon: Server },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3 text-center">
              <div className={`text-2xl font-bold ${item.color}`}>{item.value}</div>
              <div className="text-xs text-muted mt-1">{item.label}</div>
            </div>
          ))}
        </div>
      </div>

      {/* 异常状态 */}
      {(detail.unavailable > 0 || detail.misscheduled > 0) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.daemonset.abnormalStatus}</h3>
          <div className="grid grid-cols-2 gap-3">
            {detail.unavailable > 0 && (
              <div className="bg-red-50 dark:bg-red-900/20 rounded-lg p-4 flex items-center gap-3">
                <XCircle className="w-8 h-8 text-red-500" />
                <div>
                  <div className="text-2xl font-bold text-red-500">{detail.unavailable}</div>
                  <div className="text-xs text-red-600 dark:text-red-400">{t.daemonset.unavailable}</div>
                </div>
              </div>
            )}
            {detail.misscheduled > 0 && (
              <div className="bg-yellow-50 dark:bg-yellow-900/20 rounded-lg p-4 flex items-center gap-3">
                <AlertTriangle className="w-8 h-8 text-yellow-500" />
                <div>
                  <div className="text-2xl font-bold text-yellow-500">{detail.misscheduled}</div>
                  <div className="text-xs text-yellow-600 dark:text-yellow-400">{t.daemonset.misscheduled}</div>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.daemonset.basicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {[
            { label: t.common.name, value: detail.name },
            { label: t.common.namespace, value: detail.namespace },
            { label: "Age", value: detail.age || "-" },
            { label: t.daemonset.updated, value: detail.updatedScheduled },
            { label: t.common.createdAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
          ].map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium truncate">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* Conditions */}
      {detail.conditions && detail.conditions.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Conditions</h3>
          <div className="space-y-2">
            {detail.conditions.map((c, i) => (
              <div key={i} className="flex items-start gap-3 p-3 bg-[var(--background)] rounded-lg">
                {c.status === "True" ? (
                  <CheckCircle className="w-4 h-4 text-green-500 mt-0.5" />
                ) : (
                  <AlertTriangle className="w-4 h-4 text-yellow-500 mt-0.5" />
                )}
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <span className="font-medium text-default">{c.type}</span>
                    <StatusBadge status={c.status} type={c.status === "True" ? "success" : "warning"} />
                  </div>
                  {c.message && (
                    <p className="text-sm text-muted mt-1 break-words">{c.message}</p>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

// 容器 Tab
export function ContainersTab({ detail, t }: TabProps) {
  const containers = detail.template?.containers || [];

  if (containers.length === 0) {
    return <div className="text-center py-8 text-muted">{t.daemonset.noContainers}</div>;
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
              <span className="text-xs text-muted">{t.daemonset.image}: </span>
              <span className="text-sm font-mono text-default break-all">{c.image}</span>
            </div>

            {/* 端口 */}
            {c.ports && c.ports.length > 0 && (
              <div>
                <span className="text-xs text-muted block mb-1">{t.daemonset.ports}:</span>
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

// 策略 Tab
export function StrategyTab({ detail, t }: TabProps) {
  const strategy = detail.spec.updateStrategy;

  return (
    <div className="space-y-6">
      {/* 更新策略 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.daemonset.updateStrategy}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">{t.daemonset.strategyType}</div>
            <div className="text-sm font-medium text-default">
              {strategy?.type || "RollingUpdate"}
            </div>
          </div>
          {strategy?.maxUnavailable && (
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">Max Unavailable</div>
              <div className="text-sm font-medium text-default">{strategy.maxUnavailable}</div>
            </div>
          )}
          {strategy?.maxSurge && (
            <div className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">Max Surge</div>
              <div className="text-sm font-medium text-default">{strategy.maxSurge}</div>
            </div>
          )}
        </div>
      </div>

      {/* 其他配置 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.daemonset.otherConfig}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Min Ready Seconds</div>
            <div className="text-sm font-medium text-default">
              {detail.spec.minReadySeconds || 0}
            </div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-3">
            <div className="text-xs text-muted mb-1">Revision History Limit</div>
            <div className="text-sm font-medium text-default">
              {detail.spec.revisionHistoryLimit ?? 10}
            </div>
          </div>
        </div>
      </div>

      {/* 调度信息 */}
      {(detail.template?.nodeSelector || detail.template?.tolerations?.length) && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">{t.daemonset.scheduling}</h3>
          <div className="space-y-3">
            {detail.template.nodeSelector && Object.keys(detail.template.nodeSelector).length > 0 && (
              <div className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-xs text-muted mb-2">Node Selector</div>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(detail.template.nodeSelector).map(([k, v]) => (
                    <span key={k} className="px-2 py-1 bg-[var(--card-background)] rounded text-xs font-mono">
                      {k}={v}
                    </span>
                  ))}
                </div>
              </div>
            )}
            {detail.template.tolerations && detail.template.tolerations.length > 0 && (
              <div className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-xs text-muted mb-2">Tolerations ({detail.template.tolerations.length})</div>
                <div className="space-y-1">
                  {detail.template.tolerations.slice(0, 5).map((tol, i) => (
                    <div key={i} className="text-xs font-mono text-default">
                      {tol.key || "*"} {tol.operator || "Equal"} {tol.value || ""} : {tol.effect || "All"}
                    </div>
                  ))}
                  {detail.template.tolerations.length > 5 && (
                    <div className="text-xs text-muted">... {t.daemonset.moreItems.replace("{count}", String(detail.template.tolerations.length - 5))}</div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// 标签 Tab
export function LabelsTab({ detail, t }: TabProps) {
  const labels = Object.entries(detail.labels || {});
  const annotations = Object.entries(detail.annotations || {});

  return (
    <div className="space-y-6">
      {/* Labels */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Labels ({labels.length})</h3>
        {labels.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.daemonset.noLabels}</div>
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

      {/* Annotations */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">Annotations ({annotations.length})</h3>
        {annotations.length === 0 ? (
          <div className="text-center py-4 text-muted bg-[var(--background)] rounded-lg">{t.daemonset.noAnnotations}</div>
        ) : (
          <div className="space-y-2">
            {annotations.map(([key, value]) => (
              <div key={key} className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-sm font-mono text-primary break-all mb-1">{key}</div>
                <div className="text-sm text-muted break-all">{value || '""'}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
