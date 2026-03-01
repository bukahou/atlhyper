"use client";

import {
  CheckCircle2,
  XCircle,
  AlertCircle,
  AlertTriangle,
} from "lucide-react";
import type { ServiceHealth } from "@/types/model/observe";
import type { HealthStatus } from "@/types/model/apm";
import type { ObserveLandingTranslations } from "@/types/i18n";

const statusConfig: Record<HealthStatus, { icon: typeof CheckCircle2; color: string; dot: string }> = {
  healthy: { icon: CheckCircle2, color: "text-green-500", dot: "bg-green-500" },
  warning: { icon: AlertCircle, color: "text-yellow-500", dot: "bg-yellow-500" },
  critical: { icon: XCircle, color: "text-red-500", dot: "bg-red-500" },
  unknown: { icon: AlertTriangle, color: "text-muted", dot: "bg-gray-400" },
};

interface ServiceTableProps {
  services: ServiceHealth[];
  tl: ObserveLandingTranslations;
  onSelectService: (name: string) => void;
}

export function ServiceTable({ services, tl, onSelectService }: ServiceTableProps) {
  return (
    <div className="rounded-xl border border-[var(--border-color)] bg-card overflow-hidden">
      {/* Header */}
      <div className="hidden sm:grid sm:grid-cols-[auto_1fr_100px_60px_70px_80px_70px_70px] gap-x-3 px-4 py-2.5 text-[11px] font-medium text-muted uppercase tracking-wider border-b border-[var(--border-color)] bg-secondary/50">
        <span className="w-5" />
        <span>{tl.serviceHealth}</span>
        <span>{tl.namespace}</span>
        <span>{tl.replicas}</span>
        <span>{tl.rps}</span>
        <span>{tl.successRate}</span>
        <span>{tl.errorCount}</span>
        <span>{tl.p99}</span>
      </div>
      {/* Rows */}
      {services.map((svc) => {
        const cfg = statusConfig[svc.status] ?? statusConfig.unknown;
        return (
          <button
            key={svc.name}
            onClick={() => onSelectService(svc.name)}
            className="w-full text-left grid grid-cols-[auto_1fr] sm:grid-cols-[auto_1fr_100px_60px_70px_80px_70px_70px] gap-x-3 px-4 py-2.5 items-center border-b border-[var(--border-color)] last:border-b-0 hover:bg-secondary/50 transition-colors cursor-pointer"
          >
            {/* Status dot */}
            <span className={`w-2 h-2 rounded-full ${cfg.dot} flex-shrink-0`} />
            {/* Name */}
            <span className="text-sm font-medium text-default truncate">{svc.name}</span>
            {/* Namespace */}
            <span className="hidden sm:block text-xs text-muted font-mono truncate">{svc.namespace}</span>
            {/* Replicas */}
            <span className="hidden sm:block text-xs text-default font-mono">{svc.deployment?.replicas ?? "-"}</span>
            {/* RPS */}
            <span className="hidden sm:block text-xs text-default font-mono">{svc.apm?.rps.toFixed(1) ?? "-"}</span>
            {/* Success Rate */}
            <span className={`hidden sm:block text-xs font-mono font-medium ${
              (svc.apm?.successRate ?? 1) < 0.95 ? "text-red-500" : (svc.apm?.successRate ?? 1) < 0.99 ? "text-yellow-500" : "text-green-500"
            }`}>
              {svc.apm ? `${(svc.apm.successRate * 100).toFixed(2)}%` : "-"}
            </span>
            {/* Error Count */}
            <span className={`hidden sm:block text-xs font-mono ${
              (svc.logs?.errorCount ?? 0) > 100 ? "text-red-500" : (svc.logs?.errorCount ?? 0) > 10 ? "text-yellow-500" : "text-default"
            }`}>
              {svc.logs?.errorCount.toLocaleString() ?? "-"}
            </span>
            {/* P99 */}
            <span className="hidden sm:block text-xs text-default font-mono">
              {svc.apm ? `${svc.apm.p99Ms}ms` : "-"}
            </span>
          </button>
        );
      })}
    </div>
  );
}
