"use client";

import type { useI18n } from "@/i18n/context";

interface LabelsTabProps {
  labels: Record<string, string>;
  annotations: Record<string, string>;
  t: ReturnType<typeof useI18n>["t"];
}

export function LabelsTab({ labels, annotations, t }: LabelsTabProps) {
  const labelEntries = Object.entries(labels);
  const annoEntries = Object.entries(annotations);

  return (
    <div className="space-y-6">
      {/* Labels */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.labels} ({labelEntries.length})</h3>
        {labelEntries.length === 0 ? (
          <div className="text-center py-4 text-muted">{t.deployment.noLabels}</div>
        ) : (
          <div className="space-y-2">
            {labelEntries.map(([key, value]) => (
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
        <h3 className="text-sm font-semibold text-default mb-3">{t.deployment.annotations} ({annoEntries.length})</h3>
        {annoEntries.length === 0 ? (
          <div className="text-center py-4 text-muted">{t.deployment.noAnnotations}</div>
        ) : (
          <div className="space-y-2">
            {annoEntries.map(([key, value]) => (
              <div key={key} className="bg-[var(--background)] rounded-lg p-3">
                <div className="text-sm font-mono text-primary break-all mb-1">{key}</div>
                <div className="text-sm font-mono text-default break-all whitespace-pre-wrap">{value || '""'}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
