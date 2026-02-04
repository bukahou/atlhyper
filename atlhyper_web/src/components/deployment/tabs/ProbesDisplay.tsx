"use client";

import type { ProbeSpec } from "@/types/cluster";
import type { useI18n } from "@/i18n/context";

interface ProbesDisplayProps {
  liveness?: ProbeSpec;
  readiness?: ProbeSpec;
  startup?: ProbeSpec;
  t: ReturnType<typeof useI18n>["t"];
}

export function ProbesDisplay({ liveness, readiness, startup, t }: ProbesDisplayProps) {
  const probes = [
    { name: "Liveness", probe: liveness },
    { name: "Readiness", probe: readiness },
    { name: "Startup", probe: startup },
  ].filter((p) => p.probe);

  if (probes.length === 0) return null;

  return (
    <div className="mt-3">
      <div className="text-xs text-muted mb-2">{t.deployment.probes}</div>
      <div className="grid grid-cols-1 md:grid-cols-3 gap-2">
        {probes.map(({ name, probe }) => (
          <div key={name} className="bg-card rounded p-2">
            <div className="text-xs font-medium text-default mb-1">{name}</div>
            <div className="text-xs text-muted">
              {probe!.type === "httpGet" && `HTTP GET ${probe!.path || "/"}:${probe!.port}`}
              {probe!.type === "tcpSocket" && `TCP :${probe!.port}`}
              {probe!.type === "exec" && `Exec: ${probe!.command}`}
            </div>
            <div className="text-xs text-muted mt-1">
              delay={probe!.initialDelaySeconds || 0}s period={probe!.periodSeconds || 10}s
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
