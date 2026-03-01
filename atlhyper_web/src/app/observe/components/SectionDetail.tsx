"use client";

import { Lock } from "lucide-react";
import type { ServiceHealth } from "@/types/model/observe";
import type { ObserveLandingTranslations } from "@/types/i18n";

type DetailSection = "k8s" | "apm" | "slo" | "logs" | "infra";

interface SectionDetailProps {
  service: ServiceHealth;
  section: DetailSection;
  tl: ObserveLandingTranslations;
  totalLabel: string;
}

export function SectionDetail({ service, section, tl, totalLabel }: SectionDetailProps) {
  switch (section) {
    case "k8s":
      return <K8sDetail service={service} tl={tl} />;
    case "apm":
      return <ApmDetail service={service} tl={tl} />;
    case "slo":
      return <SloDetail service={service} tl={tl} />;
    case "logs":
      return <LogsDetail service={service} tl={tl} totalLabel={totalLabel} />;
    case "infra":
      return <InfraDetail service={service} tl={tl} />;
  }
}

export function sectionTitle(section: DetailSection, tl: ObserveLandingTranslations): string {
  switch (section) {
    case "k8s": return tl.k8sResources;
    case "apm": return tl.apmSection;
    case "slo": return tl.sloSection;
    case "logs": return tl.logsSection;
    case "infra": return tl.infraSection;
  }
}

// ============================================================================
// K8s Resources Detail
// ============================================================================

function K8sDetail({ service, tl }: { service: ServiceHealth; tl: ObserveLandingTranslations }) {
  const { deployment, pods, k8sService, ingresses } = service;

  return (
    <div className="space-y-4">
      {/* Deployment */}
      <DetailCard title={tl.deploymentSection}>
        {deployment ? (
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <KV label={tl.replicas} value={deployment.replicas} />
            <KV label={tl.strategy} value={deployment.strategy} />
            <KV label={tl.age} value={deployment.age} />
            <KV label={tl.image} value={deployment.image} mono />
          </div>
        ) : <NoData />}
      </DetailCard>

      {/* K8s Service */}
      {k8sService && (
        <DetailCard title="Service">
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            <KV label={tl.serviceType} value={k8sService.type} />
            <KV label={tl.clusterIP} value={k8sService.clusterIP} mono />
            <KV label={tl.ports} value={k8sService.ports} mono />
          </div>
        </DetailCard>
      )}

      {/* Pods */}
      <DetailCard title={tl.pods}>
        {pods && pods.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-muted text-left border-b border-[var(--border-color)]">
                  <th className="py-1.5 pr-3 font-medium">{tl.serviceHealth}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.phase}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.ready}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.restarts}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.node}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.age}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.cpu}</th>
                  <th className="py-1.5 font-medium">{tl.memory}</th>
                </tr>
              </thead>
              <tbody>
                {pods.map((pod) => (
                  <tr key={pod.name} className="border-b border-[var(--border-color)] last:border-b-0">
                    <td className="py-1.5 pr-3 text-default font-mono truncate max-w-[200px]">{pod.name}</td>
                    <td className="py-1.5 pr-3">
                      <span className={pod.phase === "Running" ? "text-green-500" : pod.phase === "Pending" ? "text-yellow-500" : "text-red-500"}>
                        {pod.phase}
                      </span>
                    </td>
                    <td className="py-1.5 pr-3 text-default">{pod.ready}</td>
                    <td className="py-1.5 pr-3">
                      <span className={pod.restarts > 0 ? "text-yellow-500" : "text-default"}>{pod.restarts}</span>
                    </td>
                    <td className="py-1.5 pr-3 text-muted">{pod.nodeName}</td>
                    <td className="py-1.5 pr-3 text-muted">{pod.age}</td>
                    <td className="py-1.5 pr-3 text-default font-mono">{pod.cpuUsage}</td>
                    <td className="py-1.5 text-default font-mono">{pod.memoryUsage}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : <NoData />}
      </DetailCard>

      {/* Ingress */}
      <DetailCard title={tl.ingressSection}>
        {ingresses && ingresses.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-muted text-left border-b border-[var(--border-color)]">
                  <th className="py-1.5 pr-3 font-medium">{tl.serviceHealth}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.host}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.path}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.tls}</th>
                  <th className="py-1.5 font-medium">{tl.backend}</th>
                </tr>
              </thead>
              <tbody>
                {ingresses.map((ing) => (
                  <tr key={ing.name} className="border-b border-[var(--border-color)] last:border-b-0">
                    <td className="py-1.5 pr-3 text-default font-mono">{ing.name}</td>
                    <td className="py-1.5 pr-3 text-default">{ing.hosts.join(", ")}</td>
                    <td className="py-1.5 pr-3 text-muted font-mono">
                      {ing.paths.map(p => p.path).join(", ")}
                    </td>
                    <td className="py-1.5 pr-3">
                      <span className={ing.tlsEnabled ? "text-green-500" : "text-muted"}>
                        {ing.tlsEnabled ? "✓" : "✗"}
                      </span>
                    </td>
                    <td className="py-1.5 text-muted font-mono">
                      {ing.paths.map(p => `${p.serviceName}:${p.port}`).join(", ")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : <NoData />}
      </DetailCard>
    </div>
  );
}

// ============================================================================
// APM Detail
// ============================================================================

function ApmDetail({ service, tl }: { service: ServiceHealth; tl: ObserveLandingTranslations }) {
  const apm = service.apm;
  if (!apm) return <NoData />;

  return (
    <div className="space-y-4">
      {/* Top metrics */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <MetricBox label={tl.rps} value={apm.rps.toFixed(1)} />
        <MetricBox
          label={tl.successRate}
          value={`${(apm.successRate * 100).toFixed(2)}%`}
          color={apm.successRate < 0.95 ? "text-red-500" : apm.successRate < 0.99 ? "text-yellow-500" : "text-green-500"}
        />
        <MetricBox
          label={tl.errorRate}
          value={`${(apm.errorRate * 100).toFixed(2)}%`}
          color={apm.errorRate > 0.05 ? "text-red-500" : "text-default"}
        />
        <MetricBox label={tl.p99} value={`${apm.p99Ms}ms`} />
      </div>
      {/* All metrics */}
      <DetailCard title={tl.allMetrics}>
        <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <KV label={tl.avgLatency} value={`${apm.avgMs}ms`} />
          <KV label={tl.spanCount} value={apm.spanCount.toLocaleString()} />
          <KV label={tl.errorSpanCount} value={apm.errorCount.toLocaleString()} />
        </div>
      </DetailCard>
    </div>
  );
}

// ============================================================================
// SLO Detail
// ============================================================================

function SloDetail({ service, tl }: { service: ServiceHealth; tl: ObserveLandingTranslations }) {
  const slo = service.slo;
  if (!slo) return <NoData />;

  return (
    <div className="space-y-4">
      {/* Mesh */}
      <DetailCard title={tl.meshDetail}>
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          {slo.meshSuccessRate != null && (
            <KV
              label={tl.successRate}
              value={`${(slo.meshSuccessRate * 100).toFixed(2)}%`}
              valueColor={slo.meshSuccessRate < 0.95 ? "text-red-500" : slo.meshSuccessRate < 0.99 ? "text-yellow-500" : "text-green-500"}
            />
          )}
          {slo.meshRps != null && <KV label={tl.rps} value={slo.meshRps.toFixed(1)} />}
          {slo.meshP99Ms != null && <KV label={tl.p99} value={`${slo.meshP99Ms}ms`} />}
          <div className="flex items-center gap-1.5 text-xs">
            <Lock className="w-3 h-3 text-green-500" />
            <span className={slo.mtlsEnabled ? "text-green-500" : "text-muted"}>
              mTLS {slo.mtlsEnabled ? "✓" : "✗"}
            </span>
          </div>
        </div>
      </DetailCard>

      {/* Ingress Domains */}
      {slo.ingressDomains && slo.ingressDomains.length > 0 && (
        <DetailCard title={tl.ingressSLO}>
          <div className="space-y-2">
            {slo.ingressDomains.map((d) => (
              <div key={d.domain} className="flex items-center justify-between p-2 rounded-lg bg-secondary/30">
                <span className="text-xs font-mono text-default">{d.domain}</span>
                <div className="flex items-center gap-4 text-xs">
                  <span className="text-muted">{tl.rps}: {d.rps.toFixed(1)}</span>
                  <span className="text-muted">{tl.p99}: {d.p99Ms}ms</span>
                  <span className={d.successRate < 0.95 ? "text-red-500 font-medium" : d.successRate < 0.99 ? "text-yellow-500" : "text-green-500"}>
                    {(d.successRate * 100).toFixed(2)}%
                  </span>
                </div>
              </div>
            ))}
          </div>
        </DetailCard>
      )}
    </div>
  );
}

// ============================================================================
// Logs Detail
// ============================================================================

function LogsDetail({ service, tl, totalLabel }: { service: ServiceHealth; tl: ObserveLandingTranslations; totalLabel: string }) {
  const logs = service.logs;
  if (!logs) return <NoData />;

  const total = logs.totalCount || 1;
  const errPct = ((logs.errorCount / total) * 100).toFixed(1);
  const warnPct = ((logs.warnCount / total) * 100).toFixed(1);

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-3 gap-3">
        <MetricBox
          label={tl.errorCount}
          value={logs.errorCount.toLocaleString()}
          sub={`${errPct}%`}
          color={logs.errorCount > 100 ? "text-red-500" : "text-default"}
        />
        <MetricBox
          label={tl.warnCount}
          value={logs.warnCount.toLocaleString()}
          sub={`${warnPct}%`}
          color="text-yellow-500"
        />
        <MetricBox label={totalLabel} value={logs.totalCount.toLocaleString()} />
      </div>

      {/* Distribution bar */}
      <DetailCard title={tl.distribution}>
        <div className="h-3 rounded-full overflow-hidden flex bg-secondary/50">
          <div className="bg-red-500 h-full" style={{ width: `${errPct}%` }} />
          <div className="bg-yellow-500 h-full" style={{ width: `${warnPct}%` }} />
          <div className="bg-green-500/30 h-full flex-1" />
        </div>
        <div className="flex items-center gap-4 mt-2 text-[11px] text-muted">
          <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-red-500" />{tl.errorCount} {errPct}%</span>
          <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-yellow-500" />{tl.warnCount} {warnPct}%</span>
          <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-green-500/30" />Info</span>
        </div>
      </DetailCard>
    </div>
  );
}

// ============================================================================
// Infra Detail
// ============================================================================

function InfraDetail({ service, tl }: { service: ServiceHealth; tl: ObserveLandingTranslations }) {
  const infra = service.infra;
  if (!infra) return <NoData />;

  return (
    <div className="space-y-4">
      <MetricBox label={tl.podCount} value={String(infra.podCount)} />

      {infra.nodes.map((node) => (
        <DetailCard key={node.name} title={node.name}>
          <div className="space-y-3">
            <ProgressRow label={tl.cpu} value={node.cpuPct} />
            <ProgressRow label={tl.memory} value={node.memPct} />
          </div>
        </DetailCard>
      ))}
    </div>
  );
}

// ============================================================================
// Shared sub-components
// ============================================================================

function DetailCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-[var(--border-color)] bg-card p-3">
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2.5">{title}</h4>
      {children}
    </div>
  );
}

function KV({ label, value, mono, valueColor }: { label: string; value: string; mono?: boolean; valueColor?: string }) {
  return (
    <div>
      <p className="text-[11px] text-muted mb-0.5">{label}</p>
      <p className={`text-xs ${mono ? "font-mono" : ""} ${valueColor ?? "text-default"} truncate`}>{value}</p>
    </div>
  );
}

function MetricBox({ label, value, sub, color }: { label: string; value: string; sub?: string; color?: string }) {
  return (
    <div className="p-3 rounded-xl border border-[var(--border-color)] bg-card">
      <p className="text-xs text-muted mb-1">{label}</p>
      <p className={`text-xl font-bold ${color ?? "text-default"}`}>{value}</p>
      {sub && <p className="text-[11px] text-muted mt-0.5">{sub}</p>}
    </div>
  );
}

function ProgressRow({ label, value }: { label: string; value: number }) {
  const color = value > 85 ? "bg-red-500" : value > 70 ? "bg-yellow-500" : "bg-green-500";
  return (
    <div>
      <div className="flex items-center justify-between text-xs mb-1">
        <span className="text-muted">{label}</span>
        <span className="text-default font-mono">{value.toFixed(1)}%</span>
      </div>
      <div className="h-2 rounded-full bg-secondary/50 overflow-hidden">
        <div className={`h-full rounded-full ${color} transition-all`} style={{ width: `${value}%` }} />
      </div>
    </div>
  );
}

function NoData() {
  return <p className="text-xs text-muted py-2">-</p>;
}
