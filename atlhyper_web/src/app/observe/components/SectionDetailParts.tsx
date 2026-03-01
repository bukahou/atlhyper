"use client";

// Shared sub-components for SectionDetail

export function DetailCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-[var(--border-color)] bg-card p-3">
      <h4 className="text-xs font-semibold text-muted uppercase tracking-wider mb-2.5">{title}</h4>
      {children}
    </div>
  );
}

export function KV({ label, value, mono, valueColor }: { label: string; value: string; mono?: boolean; valueColor?: string }) {
  return (
    <div>
      <p className="text-[11px] text-muted mb-0.5">{label}</p>
      <p className={`text-xs ${mono ? "font-mono" : ""} ${valueColor ?? "text-default"} truncate`}>{value}</p>
    </div>
  );
}

export function MetricBox({ label, value, sub, color }: { label: string; value: string; sub?: string; color?: string }) {
  return (
    <div className="p-3 rounded-xl border border-[var(--border-color)] bg-card">
      <p className="text-xs text-muted mb-1">{label}</p>
      <p className={`text-xl font-bold ${color ?? "text-default"}`}>{value}</p>
      {sub && <p className="text-[11px] text-muted mt-0.5">{sub}</p>}
    </div>
  );
}

export function ProgressRow({ label, value }: { label: string; value: number }) {
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

export function NoData() {
  return <p className="text-xs text-muted py-2">-</p>;
}
