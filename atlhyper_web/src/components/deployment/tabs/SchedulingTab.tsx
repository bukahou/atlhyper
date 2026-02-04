"use client";

import type { DeploymentDetail, TolerationSpec } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";
import { InfoCard } from "./InfoCard";

interface SchedulingTabProps {
  detail: DeploymentDetail;
  t: ReturnType<typeof useI18n>["t"];
}

export function SchedulingTab({ detail, t }: SchedulingTabProps) {
  const template = detail.template;
  return (
    <div className="space-y-6">
      {/* 基本调度配置 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.schedulingConfig}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          <InfoCard label="ServiceAccount" value={template?.serviceAccountName || "default"} />
          <InfoCard label="RuntimeClass" value={template?.runtimeClassName || "-"} />
          <InfoCard label="DNSPolicy" value={template?.dnsPolicy || "ClusterFirst"} />
          <InfoCard label={t.deployment.hostNetwork} value={template?.hostNetwork ? t.common.yes : t.common.no} />
        </div>
      </div>

      {/* NodeSelector */}
      {template?.nodeSelector && Object.keys(template.nodeSelector).length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">NodeSelector</h3>
          <div className="space-y-2">
            {Object.entries(template.nodeSelector).map(([k, v]) => (
              <div key={k} className="bg-[var(--background)] rounded-lg p-3 flex items-start gap-2">
                <span className="text-sm font-mono text-primary">{k}</span>
                <span className="text-muted">=</span>
                <span className="text-sm font-mono text-default">{v}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Tolerations */}
      {template?.tolerations && template.tolerations.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Tolerations</h3>
          <div className="space-y-2">
            {template.tolerations.map((tol: TolerationSpec, i: number) => (
              <div key={i} className="bg-[var(--background)] rounded-lg p-3">
                <div className="flex items-center gap-2 flex-wrap">
                  {tol.key && <span className="text-sm font-mono text-primary">{tol.key}</span>}
                  {tol.operator && <span className="text-xs text-muted">{tol.operator}</span>}
                  {tol.value && <span className="text-sm font-mono text-default">{tol.value}</span>}
                  {tol.effect && (
                    <span className="px-2 py-0.5 bg-card text-xs rounded">{tol.effect}</span>
                  )}
                  {tol.tolerationSeconds !== undefined && (
                    <span className="text-xs text-muted">({tol.tolerationSeconds}s)</span>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Affinity */}
      {template?.affinity && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Affinity</h3>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {template.affinity.nodeAffinity && <InfoCard label="NodeAffinity" value={template.affinity.nodeAffinity} />}
            {template.affinity.podAffinity && <InfoCard label="PodAffinity" value={template.affinity.podAffinity} />}
            {template.affinity.podAntiAffinity && <InfoCard label="PodAntiAffinity" value={template.affinity.podAntiAffinity} />}
          </div>
        </div>
      )}

      {/* ImagePullSecrets */}
      {template?.imagePullSecrets && template.imagePullSecrets.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">ImagePullSecrets</h3>
          <div className="flex flex-wrap gap-2">
            {template.imagePullSecrets.map((s, i) => (
              <span key={i} className="px-2 py-1 bg-[var(--background)] text-sm font-mono rounded">{s}</span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
