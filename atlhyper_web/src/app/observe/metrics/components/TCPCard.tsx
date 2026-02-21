import { Network } from "lucide-react";
import type { TCPMetrics, SoftnetMetrics } from "@/types/node-metrics";
import { useI18n } from "@/i18n/context";

export function TCPCard({ tcp, softnet }: { tcp: TCPMetrics; softnet: SoftnetMetrics }) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const states = [
    { label: "ESTABLISHED", value: tcp.currEstab, color: "text-green-500" },
    { label: "TIME_WAIT", value: tcp.timeWait, color: tcp.timeWait > 200 ? "text-yellow-500" : "text-default" },
    { label: "ORPHAN", value: tcp.orphan, color: tcp.orphan > 0 ? "text-yellow-500" : "text-default" },
  ];

  const total = tcp.currEstab + tcp.timeWait + tcp.orphan;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 sm:p-2 bg-blue-500/10 rounded-lg">
            <Network className="w-4 h-4 sm:w-5 sm:h-5 text-blue-500" />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">{nm.tcp.title}</h3>
            <p className="text-[10px] sm:text-xs text-muted">{total} {nm.tcp.activeConnections}</p>
          </div>
        </div>
      </div>

      {/* TCP States */}
      <div className="grid grid-cols-3 gap-2 mb-3">
        {states.map((s) => (
          <div key={s.label} className="p-2 bg-[var(--background)] rounded-lg text-center">
            <div className={`text-base sm:text-lg font-bold ${s.color}`}>{s.value}</div>
            <div className="text-[9px] sm:text-[10px] text-muted leading-tight mt-0.5">{s.label}</div>
          </div>
        ))}
      </div>

      {/* Stacked bar */}
      {total > 0 && (
        <div className="h-2.5 rounded-full overflow-hidden flex mb-3">
          {tcp.currEstab > 0 && <div className="h-full bg-green-500" style={{ width: `${(tcp.currEstab / total) * 100}%` }} />}
          {tcp.timeWait > 0 && <div className="h-full bg-yellow-500" style={{ width: `${(tcp.timeWait / total) * 100}%` }} />}
          {tcp.orphan > 0 && <div className="h-full bg-red-500" style={{ width: `${(tcp.orphan / total) * 100}%` }} />}
        </div>
      )}

      {/* Socket stats */}
      <div className="grid grid-cols-3 gap-2 text-xs">
        <div className="p-2 bg-[var(--background)] rounded-lg">
          <div className="text-muted text-[10px]">{nm.tcp.alloc}</div>
          <div className="font-medium text-default">{tcp.alloc}</div>
        </div>
        <div className="p-2 bg-[var(--background)] rounded-lg">
          <div className="text-muted text-[10px]">{nm.tcp.inUse}</div>
          <div className="font-medium text-default">{tcp.inUse}</div>
        </div>
        <div className="p-2 bg-[var(--background)] rounded-lg">
          <div className="text-muted text-[10px]">{nm.tcp.socketsUsed}</div>
          <div className="font-medium text-default">{tcp.socketsUsed}</div>
        </div>
      </div>

      {/* Softnet */}
      <div className="mt-3 pt-3 border-t border-[var(--border-color)] grid grid-cols-2 gap-2 text-xs">
        <div className="flex items-center justify-between p-2 bg-[var(--background)] rounded-lg">
          <span className="text-muted">{nm.tcp.softnetDropped}</span>
          <span className={`font-medium ${softnet.dropped > 0 ? "text-red-500" : "text-green-500"}`}>{softnet.dropped}</span>
        </div>
        <div className="flex items-center justify-between p-2 bg-[var(--background)] rounded-lg">
          <span className="text-muted">{nm.tcp.softnetSqueezed}</span>
          <span className={`font-medium ${softnet.squeezed > 50 ? "text-yellow-500" : "text-default"}`}>{softnet.squeezed}</span>
        </div>
      </div>
    </div>
  );
}
