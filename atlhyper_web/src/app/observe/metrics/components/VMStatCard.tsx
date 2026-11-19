import { MemoryStick, AlertTriangle } from "lucide-react";
import type { NodeVMStat, NodeVMStatLevels } from "@/types/node-metrics";
import { useI18n } from "@/i18n/context";
import { hardwareStatusText } from "@/lib/hardware-status";

const fmtN = (n: number) => n >= 1e6 ? (n / 1e6).toFixed(1) + "M" : n >= 1e3 ? (n / 1e3).toFixed(1) + "K" : n.toFixed(1);

/** 读数正常时用默认文字色，有等级时按等级上色 */
const valueColor = (level: NodeVMStatLevels[keyof NodeVMStatLevels]) =>
  level === "good" ? "text-default" : hardwareStatusText(level);

export function VMStatCard({ data, levels }: { data: NodeVMStat; levels: NodeVMStatLevels }) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const swapActive = levels.swapIn === "warn" || levels.swapIn === "crit" || levels.swapOut === "warn" || levels.swapOut === "crit";
  const majorFaultAlarm = levels.majorFaults === "warn" || levels.majorFaults === "crit";

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 sm:p-2 bg-violet-500/10 rounded-lg">
          <MemoryStick className="w-4 h-4 sm:w-5 sm:h-5 text-violet-500" />
        </div>
        <div>
          <h3 className="text-sm sm:text-base font-semibold text-default">{nm.vmstat.title}</h3>
          <p className="text-[10px] sm:text-xs text-muted">{nm.vmstat.description}</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-2 sm:gap-3">
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">{nm.vmstat.pageFaults}</div>
          <div className="text-base sm:text-lg font-bold text-default">{fmtN(data.pgFaultPerSec)}</div>
          <div className="text-[10px] text-muted">{nm.vmstat.perSecond}</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">{nm.vmstat.majorFaults}</div>
          <div className={`text-base sm:text-lg font-bold ${valueColor(levels.majorFaults)}`}>{fmtN(data.pgMajFaultPerSec)}</div>
          <div className="text-[10px] text-muted">{nm.vmstat.perSecondDisk}</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">{nm.vmstat.swapIn}</div>
          <div className={`text-base sm:text-lg font-bold ${valueColor(levels.swapIn)}`}>{fmtN(data.pswpInPerSec)}</div>
          <div className="text-[10px] text-muted">{nm.vmstat.pagesPerSec}</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-1">{nm.vmstat.swapOut}</div>
          <div className={`text-base sm:text-lg font-bold ${valueColor(levels.swapOut)}`}>{fmtN(data.pswpOutPerSec)}</div>
          <div className="text-[10px] text-muted">{nm.vmstat.pagesPerSec}</div>
        </div>
      </div>

      {(swapActive || majorFaultAlarm) && (
        <div className={`mt-3 pt-3 border-t border-[var(--border-color)] flex items-center gap-2 ${hardwareStatusText(majorFaultAlarm ? levels.majorFaults : "warn")}`}>
          <AlertTriangle className="w-3.5 h-3.5 flex-shrink-0" />
          <span className="text-xs">
            {majorFaultAlarm ? nm.vmstat.highMajorFaults : nm.vmstat.activeSwapIO}
          </span>
        </div>
      )}
    </div>
  );
}
