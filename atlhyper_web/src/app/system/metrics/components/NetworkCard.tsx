"use client";

import { memo } from "react";
import { Network, ArrowDownToLine, ArrowUpFromLine, Wifi, WifiOff } from "lucide-react";
import type { NetworkMetrics } from "@/types/node-metrics";
import { formatBytesPS, formatNumber } from "../mock/data";

interface NetworkCardProps {
  data: NetworkMetrics[];
}

export const NetworkCard = memo(function NetworkCard({ data }: NetworkCardProps) {
  // 计算总流量
  const totalRxPS = data.reduce((acc, n) => acc + n.rxBytesPS, 0);
  const totalTxPS = data.reduce((acc, n) => acc + n.txBytesPS, 0);
  const totalErrors = data.reduce((acc, n) => acc + n.rxErrors + n.txErrors, 0);
  const totalDropped = data.reduce((acc, n) => acc + n.rxDropped + n.txDropped, 0);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-blue-500/10 rounded-lg">
            <Network className="w-5 h-5 text-blue-500" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-default">Network</h3>
            <p className="text-xs text-muted">{data.length} interface(s)</p>
          </div>
        </div>
        {/* 总流量 */}
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-1">
            <ArrowDownToLine className="w-4 h-4 text-green-500" />
            <span className="text-sm font-medium text-default">{formatBytesPS(totalRxPS)}</span>
          </div>
          <div className="flex items-center gap-1">
            <ArrowUpFromLine className="w-4 h-4 text-blue-500" />
            <span className="text-sm font-medium text-default">{formatBytesPS(totalTxPS)}</span>
          </div>
        </div>
      </div>

      {/* 网络接口列表 */}
      <div className="space-y-3">
        {data.map((iface) => (
          <div key={iface.interface} className="p-3 bg-[var(--background)] rounded-lg">
            {/* 接口名 & 状态 */}
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                {iface.status === "up" ? (
                  <Wifi className="w-4 h-4 text-green-500" />
                ) : (
                  <WifiOff className="w-4 h-4 text-red-500" />
                )}
                <span className="text-sm font-medium text-default">{iface.interface}</span>
                <span className="text-xs text-muted">{iface.ipAddress}</span>
              </div>
              <span className="text-xs text-muted">
                {iface.speed >= 1000 ? `${iface.speed / 1000} Gbps` : `${iface.speed} Mbps`}
              </span>
            </div>

            {/* 流量 */}
            <div className="grid grid-cols-2 gap-4 mb-2">
              <div>
                <div className="flex items-center justify-between text-xs mb-1">
                  <span className="text-muted flex items-center gap-1">
                    <ArrowDownToLine className="w-3 h-3 text-green-500" />
                    Receive
                  </span>
                  <span className="text-green-500 font-medium">{formatBytesPS(iface.rxBytesPS)}</span>
                </div>
                <div className="h-1.5 bg-[var(--background-secondary,#1f2937)] rounded-full overflow-hidden">
                  <div
                    className="h-full bg-green-500 rounded-full transition-all duration-300"
                    style={{ width: `${Math.min(100, (iface.rxBytesPS / (iface.speed * 125000)) * 100)}%` }}
                  />
                </div>
              </div>
              <div>
                <div className="flex items-center justify-between text-xs mb-1">
                  <span className="text-muted flex items-center gap-1">
                    <ArrowUpFromLine className="w-3 h-3 text-blue-500" />
                    Transmit
                  </span>
                  <span className="text-blue-500 font-medium">{formatBytesPS(iface.txBytesPS)}</span>
                </div>
                <div className="h-1.5 bg-[var(--background-secondary,#1f2937)] rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-500 rounded-full transition-all duration-300"
                    style={{ width: `${Math.min(100, (iface.txBytesPS / (iface.speed * 125000)) * 100)}%` }}
                  />
                </div>
              </div>
            </div>

            {/* 包 & 错误 */}
            <div className="grid grid-cols-4 gap-2 text-xs">
              <div>
                <div className="text-muted">Rx Pkts</div>
                <div className="text-default">{formatNumber(iface.rxPacketsPS)}/s</div>
              </div>
              <div>
                <div className="text-muted">Tx Pkts</div>
                <div className="text-default">{formatNumber(iface.txPacketsPS)}/s</div>
              </div>
              <div title="自系统启动以来的累计错误数">
                <div className="text-muted">Errors</div>
                <div className="text-default">
                  {formatNumber(iface.rxErrors + iface.txErrors)}
                </div>
              </div>
              <div title="自系统启动以来的累计丢包数">
                <div className="text-muted">Dropped</div>
                <div className="text-default">
                  {formatNumber(iface.rxDropped + iface.txDropped)}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* 底部统计信息 */}
      {(totalErrors > 0 || totalDropped > 0) && (
        <div className="mt-4 pt-4 border-t border-[var(--border-color)]">
          <div className="flex items-center gap-2 text-xs text-muted">
            <span>累计统计:</span>
            {totalErrors > 0 && (
              <span>{formatNumber(totalErrors)} 错误</span>
            )}
            {totalErrors > 0 && totalDropped > 0 && <span>·</span>}
            {totalDropped > 0 && (
              <span>{formatNumber(totalDropped)} 丢包</span>
            )}
            <span className="text-muted/60">(自启动以来)</span>
          </div>
        </div>
      )}
    </div>
  );
});
