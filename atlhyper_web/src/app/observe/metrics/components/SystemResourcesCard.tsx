import { Activity, FileText, Shield, Zap } from "lucide-react";
import type { NodeSystem } from "@/types/node-metrics";
import { useI18n } from "@/i18n/context";

const usageColor = (v: number) => v >= 80 ? "text-red-500" : v >= 60 ? "text-yellow-500" : "text-emerald-500";
const usageBg = (v: number) => v >= 80 ? "bg-red-500" : v >= 60 ? "bg-yellow-500" : "bg-emerald-500";
const fmtN = (n: number) => n >= 1e6 ? (n / 1e6).toFixed(1) + "M" : n >= 1e3 ? (n / 1e3).toFixed(1) + "K" : n.toString();

export function SystemResourcesCard({ system }: { system: NodeSystem }) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const fdPct = system.filefdMax > 0 ? (system.filefdAllocated / system.filefdMax) * 100 : 0;
  const ctPct = system.conntrackLimit > 0 ? (system.conntrackEntries / system.conntrackLimit) * 100 : 0;
  const entropyOk = system.entropyBits >= 256;

  const items = [
    { label: nm.system.fileDescriptors, value: fmtN(system.filefdAllocated), pct: fdPct, max: `Max: ${fmtN(system.filefdMax)}`, icon: FileText },
    { label: nm.system.conntrackTable, value: fmtN(system.conntrackEntries), pct: ctPct, max: `Limit: ${fmtN(system.conntrackLimit)}`, icon: Shield },
  ];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 sm:p-2 bg-emerald-500/10 rounded-lg">
          <Activity className="w-4 h-4 sm:w-5 sm:h-5 text-emerald-500" />
        </div>
        <h3 className="text-sm sm:text-base font-semibold text-default">{nm.system.title}</h3>
      </div>

      <div className="space-y-3">
        {items.map((it) => (
          <div key={it.label} className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
            <div className="flex items-center justify-between mb-1.5">
              <div className="flex items-center gap-1.5">
                <it.icon className="w-3.5 h-3.5 text-muted" />
                <span className="text-xs sm:text-sm text-default">{it.label}</span>
              </div>
              <span className={`text-xs sm:text-sm font-bold ${usageColor(it.pct)}`}>{it.value}</span>
            </div>
            <div className="h-1.5 bg-[var(--card-bg)] rounded-full overflow-hidden mb-1">
              <div className={`h-full rounded-full ${usageBg(it.pct)}`} style={{ width: `${Math.min(100, it.pct)}%` }} />
            </div>
            <div className="flex justify-between text-[10px] text-muted">
              <span>{it.pct.toFixed(2)}% {nm.system.used}</span>
              <span>{it.max}</span>
            </div>
          </div>
        ))}

        {/* Entropy */}
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5">
              <Zap className="w-3.5 h-3.5 text-muted" />
              <span className="text-xs sm:text-sm text-default">{nm.system.entropyPool}</span>
            </div>
            <span className={`text-xs sm:text-sm font-bold ${entropyOk ? "text-green-500" : "text-red-500"}`}>
              {system.entropyBits} {nm.system.bits}
            </span>
          </div>
          <div className="text-[10px] text-muted mt-1">
            {entropyOk ? nm.system.entropySufficient : nm.system.entropyLow}
          </div>
        </div>
      </div>
    </div>
  );
}
