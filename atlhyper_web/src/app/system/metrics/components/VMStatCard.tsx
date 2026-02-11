import { MemoryStick, AlertTriangle } from "lucide-react";
import type { VMStatMetrics } from "@/types/node-metrics";

const fmtN = (n: number) => n >= 1e6 ? (n / 1e6).toFixed(1) + "M" : n >= 1e3 ? (n / 1e3).toFixed(1) + "K" : n.toFixed(1);

export function VMStatCard({ data }: { data: VMStatMetrics }) {
  const swapActive = data.pswpinPS > 0 || data.pswpoutPS > 0;
  const majorFaultWarn = data.pgmajfaultPS > 100;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 sm:p-2 bg-violet-500/10 rounded-lg">
          <MemoryStick className="w-4 h-4 sm:w-5 sm:h-5 text-violet-500" />
        </div>
        <div>
          <h3 className="text-sm sm:text-base font-semibold text-default">Virtual Memory</h3>
          <p className="text-[10px] sm:text-xs text-muted">Page faults & swap activity (/sec)</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 sm:gap-3">
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">Page Faults (minor)</div>
          <div className="text-base sm:text-lg font-bold text-default">{fmtN(data.pgfaultPS)}</div>
          <div className="text-[10px] text-muted">per second</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">Major Faults</div>
          <div className={`text-base sm:text-lg font-bold ${majorFaultWarn ? "text-red-500" : "text-default"}`}>{fmtN(data.pgmajfaultPS)}</div>
          <div className="text-[10px] text-muted">per second (disk read)</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">Swap In</div>
          <div className={`text-base sm:text-lg font-bold ${data.pswpinPS > 0 ? "text-yellow-500" : "text-default"}`}>{fmtN(data.pswpinPS)}</div>
          <div className="text-[10px] text-muted">pages/sec</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">Swap Out</div>
          <div className={`text-base sm:text-lg font-bold ${data.pswpoutPS > 0 ? "text-yellow-500" : "text-default"}`}>{fmtN(data.pswpoutPS)}</div>
          <div className="text-[10px] text-muted">pages/sec</div>
        </div>
      </div>

      {(swapActive || majorFaultWarn) && (
        <div className={`mt-3 pt-3 border-t border-[var(--border-color)] flex items-center gap-2 ${majorFaultWarn ? "text-red-500" : "text-yellow-500"}`}>
          <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
          <span className="text-xs">
            {majorFaultWarn ? "High major page faults — possible memory thrashing" : "Active swap I/O — memory pressure detected"}
          </span>
        </div>
      )}
    </div>
  );
}
