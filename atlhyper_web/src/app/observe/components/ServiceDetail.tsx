"use client";

import {
  ChevronRight,
  Activity,
  Shield,
  FileText,
  Server,
  Box,
  CheckCircle2,
  XCircle,
  AlertCircle,
  AlertTriangle,
} from "lucide-react";
import type { ServiceHealth } from "@/types/model/observe";
import type { HealthStatus } from "@/types/model/apm";
import type { ObserveLandingTranslations } from "@/types/i18n";

type DetailSection = "k8s" | "apm" | "slo" | "logs" | "infra";

const statusConfig: Record<HealthStatus, { icon: typeof CheckCircle2; color: string; bg: string; dot: string }> = {
  healthy: { icon: CheckCircle2, color: "text-green-500", bg: "bg-green-500/10", dot: "bg-green-500" },
  warning: { icon: AlertCircle, color: "text-yellow-500", bg: "bg-yellow-500/10", dot: "bg-yellow-500" },
  critical: { icon: XCircle, color: "text-red-500", bg: "bg-red-500/10", dot: "bg-red-500" },
  unknown: { icon: AlertTriangle, color: "text-muted", bg: "bg-gray-500/10", dot: "bg-gray-400" },
};

interface ServiceDetailProps {
  service: ServiceHealth;
  tl: ObserveLandingTranslations;
  totalLabel: string;
  onDrillDown: (section: DetailSection) => void;
}

export function ServiceDetail({ service, tl, totalLabel, onDrillDown }: ServiceDetailProps) {
  const cfg = statusConfig[service.status] ?? statusConfig.unknown;
  const { apm, slo, logs, infra, deployment, pods, ingresses } = service;

  return (
    <div className="space-y-4">
      {/* Service header */}
      <div className="flex items-center gap-3">
        <span className={`w-2.5 h-2.5 rounded-full ${cfg.dot}`} />
        <h2 className="text-base font-semibold text-default">{service.name}</h2>
        <span className="text-xs text-muted font-mono">{service.namespace}</span>
      </div>

      {/* Metric cards */}
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">
        <MetricCard label={tl.replicas} value={deployment?.replicas ?? "-"} color="text-blue-500" />
        <MetricCard label={tl.rps} value={apm?.rps.toFixed(1) ?? "-"} color="text-blue-500" />
        <MetricCard
          label={tl.successRate}
          value={apm ? `${(apm.successRate * 100).toFixed(2)}%` : "-"}
          color={!apm ? "text-muted" : apm.successRate < 0.95 ? "text-red-500" : apm.successRate < 0.99 ? "text-yellow-500" : "text-green-500"}
        />
        <MetricCard
          label={tl.errorCount}
          value={logs?.errorCount.toLocaleString() ?? "-"}
          color={(logs?.errorCount ?? 0) > 100 ? "text-red-500" : "text-default"}
        />
        <MetricCard label={tl.podCount} value={String(pods?.length ?? infra?.podCount ?? "-")} color="text-cyan-500" />
      </div>

      {/* 5 drill-down panels */}
      <div className="space-y-2">
        {/* K8s Resources */}
        <DrillPanel
          icon={Box}
          label={tl.k8sResources}
          onClick={() => onDrillDown("k8s")}
          detailLabel={tl.viewDetail}
        >
          <span className="text-xs text-secondary">
            {deployment ? `${tl.deploymentSection}: ${deployment.name}  ${tl.replicas} ${deployment.replicas}  ${tl.image} ${deployment.image}` : "-"}
          </span>
          {(pods && pods.length > 0) && (
            <span className="text-xs text-muted ml-3">
              Pod: {pods.length} {pods[0].phase}
            </span>
          )}
          {(ingresses && ingresses.length > 0) && (
            <span className="text-xs text-muted ml-3">
              {tl.ingressSection}: {ingresses.flatMap(i => i.hosts).join(", ")}
            </span>
          )}
        </DrillPanel>

        {/* APM */}
        <DrillPanel
          icon={Activity}
          label={tl.apmSection}
          onClick={() => onDrillDown("apm")}
          detailLabel={tl.viewDetail}
        >
          {apm ? (
            <span className="text-xs text-secondary">
              {tl.rps}: {apm.rps.toFixed(1)}  {tl.successRate}: {(apm.successRate * 100).toFixed(2)}%  {tl.errorRate}: {(apm.errorRate * 100).toFixed(2)}%  {tl.p99}: {apm.p99Ms}ms
            </span>
          ) : <span className="text-xs text-muted">-</span>}
        </DrillPanel>

        {/* SLO */}
        <DrillPanel
          icon={Shield}
          label={tl.sloSection}
          onClick={() => onDrillDown("slo")}
          detailLabel={tl.viewDetail}
        >
          {slo ? (
            <span className="text-xs text-secondary">
              {slo.meshSuccessRate != null && `Mesh: ${(slo.meshSuccessRate * 100).toFixed(2)}%`}
              {slo.mtlsEnabled && "  mTLS: ✓"}
              {slo.ingressDomains && slo.ingressDomains.length > 0 && `  ${tl.ingressSection}: ${slo.ingressDomains.length} ${tl.domain}`}
            </span>
          ) : <span className="text-xs text-muted">-</span>}
        </DrillPanel>

        {/* Logs */}
        <DrillPanel
          icon={FileText}
          label={tl.logsSection}
          onClick={() => onDrillDown("logs")}
          detailLabel={tl.viewDetail}
        >
          {logs ? (
            <span className="text-xs text-secondary">
              {tl.errorCount}: {logs.errorCount.toLocaleString()}  {tl.warnCount}: {logs.warnCount.toLocaleString()}  {totalLabel}: {logs.totalCount.toLocaleString()}
            </span>
          ) : <span className="text-xs text-muted">-</span>}
        </DrillPanel>

        {/* Infra */}
        <DrillPanel
          icon={Server}
          label={tl.infraSection}
          onClick={() => onDrillDown("infra")}
          detailLabel={tl.viewDetail}
        >
          {infra ? (
            <span className="text-xs text-secondary">
              Pod: {infra.podCount}  {tl.node}: {infra.nodes.map(n => `${n.name} (CPU ${n.cpuPct.toFixed(0)}%)`).join(", ")}
            </span>
          ) : <span className="text-xs text-muted">-</span>}
        </DrillPanel>
      </div>
    </div>
  );
}

function MetricCard({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div className="p-3 rounded-xl border border-[var(--border-color)] bg-card">
      <p className="text-xs text-muted mb-1">{label}</p>
      <p className={`text-xl font-bold ${color}`}>{value}</p>
    </div>
  );
}

function DrillPanel({
  icon: Icon,
  label,
  children,
  onClick,
  detailLabel,
}: {
  icon: typeof Activity;
  label: string;
  children: React.ReactNode;
  onClick: () => void;
  detailLabel: string;
}) {
  return (
    <button
      onClick={onClick}
      className="w-full text-left flex items-center gap-3 p-3 rounded-xl border border-[var(--border-color)] bg-card hover:border-primary/30 transition-colors cursor-pointer group"
    >
      <Icon className="w-4 h-4 text-muted flex-shrink-0" />
      <span className="text-xs font-semibold text-default w-20 flex-shrink-0">{label}</span>
      <div className="flex-1 min-w-0 truncate">{children}</div>
      <span className="text-xs text-primary opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-0.5 flex-shrink-0">
        {detailLabel} <ChevronRight className="w-3 h-3" />
      </span>
    </button>
  );
}
