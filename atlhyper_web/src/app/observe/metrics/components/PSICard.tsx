import { Cpu, HardDrive, MemoryStick, Timer } from "lucide-react";
import type { NodePSI } from "@/types/node-metrics";
import { useI18n } from "@/i18n/context";

const psiColor = (v: number) =>
  v >= 25 ? "text-red-500" : v >= 10 ? "text-yellow-500" : v >= 1 ? "text-blue-500" : "text-emerald-500";

const psiBg = (v: number) =>
  v >= 25 ? "bg-red-500" : v >= 10 ? "bg-yellow-500" : v >= 1 ? "bg-blue-500" : "bg-emerald-500";

export function PSICard({ data }: { data: NodePSI }) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const resources = [
    { name: "CPU", some: data.cpuSomePct, full: undefined as number | undefined, icon: Cpu, color: "text-orange-500" },
    { name: "Memory", some: data.memSomePct, full: data.memFullPct, icon: MemoryStick, color: "text-green-500" },
    { name: "I/O", some: data.ioSomePct, full: data.ioFullPct, icon: HardDrive, color: "text-purple-500" },
  ];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 sm:p-2 bg-amber-500/10 rounded-lg">
          <Timer className="w-4 h-4 sm:w-5 sm:h-5 text-amber-500" />
        </div>
        <div>
          <h3 className="text-sm sm:text-base font-semibold text-default">{nm.psi.title}</h3>
          <p className="text-[10px] sm:text-xs text-muted">{nm.psi.description}</p>
        </div>
      </div>

      <div className="space-y-3">
        {resources.map((r) => (
          <div key={r.name} className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
            <div className="flex items-center gap-2 mb-2">
              <r.icon className={`w-3.5 h-3.5 ${r.color}`} />
              <span className="text-xs sm:text-sm font-medium text-default">{r.name}</span>
              <span className={`text-xs font-bold ml-auto ${psiColor(r.some)}`}>
                {r.some.toFixed(2)}%
              </span>
            </div>
            <div className="h-1.5 bg-[var(--card-bg)] rounded-full overflow-hidden mb-1">
              <div className={`h-full rounded-full ${psiBg(r.some)}`} style={{ width: `${Math.min(100, r.some * 2)}%` }} />
            </div>
            <div className="flex items-center justify-between text-[10px] text-muted">
              <span>{nm.psi.someDesc}</span>
            </div>
            {r.full !== undefined && (
              <div className="mt-1.5 pt-1.5 border-t border-[var(--border-color)] flex items-center justify-between text-[10px]">
                <span className="text-muted">{nm.psi.fullDesc}</span>
                <span className={psiColor(r.full)}>{r.full.toFixed(2)}%</span>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
