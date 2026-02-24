"use client";

import { memo } from "react";
import { Database, ArrowDown, ArrowUp, Activity } from "lucide-react";
import type { NodeDisk } from "@/types/node-metrics";
import { formatBytes, formatBytesPS } from "@/lib/format";
import { useI18n } from "@/i18n/context";

interface DiskCardProps {
  data: NodeDisk[];
}

const getUsageColor = (usage: number) => {
  if (usage >= 80) return "bg-red-500";
  if (usage >= 60) return "bg-yellow-500";
  return "bg-green-500";
};

const getUsageTextColor = (usage: number) => {
  if (usage >= 80) return "text-red-500";
  if (usage >= 60) return "text-yellow-500";
  return "text-green-500";
};

export const DiskCard = memo(function DiskCard({ data }: DiskCardProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  // 计算总 I/O
  const totalReadPS = data.reduce((acc, d) => acc + d.readBytesPerSec, 0);
  const totalWritePS = data.reduce((acc, d) => acc + d.writeBytesPerSec, 0);
  const totalIOPS = data.reduce((acc, d) => acc + d.readIOPS + d.writeIOPS, 0);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-3 sm:mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 sm:p-2 bg-purple-500/10 rounded-lg">
            <Database className="w-4 h-4 sm:w-5 sm:h-5 text-purple-500" />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">{nm.disk.title}</h3>
            <p className="text-[10px] sm:text-xs text-muted">{data.length} {nm.disk.mounts}</p>
          </div>
        </div>
        {/* 总 I/O 速率 */}
        <div className="flex items-center gap-2 sm:gap-3">
          <div className="flex items-center gap-1 text-xs sm:text-sm">
            <ArrowDown className="w-3 h-3 text-blue-500" />
            <span className="text-default">{formatBytesPS(totalReadPS)}</span>
          </div>
          <div className="flex items-center gap-1 text-xs sm:text-sm">
            <ArrowUp className="w-3 h-3 text-orange-500" />
            <span className="text-default">{formatBytesPS(totalWritePS)}</span>
          </div>
        </div>
      </div>

      {/* 磁盘列表 */}
      <div className="space-y-3 sm:space-y-4">
        {data.map((disk, idx) => {
          const usedBytes = disk.totalBytes - disk.availBytes;
          return (
            <div key={disk.mountPoint || `disk-${idx}`} className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
              {/* 设备 & 挂载点 */}
              <div className="flex items-center justify-between mb-2">
                <div className="min-w-0 flex-1">
                  <span className="text-xs sm:text-sm font-medium text-default">{disk.mountPoint}</span>
                  <span className="text-[10px] sm:text-xs text-muted ml-1 sm:ml-2 hidden sm:inline">({disk.device})</span>
                </div>
                <span className="text-[10px] sm:text-xs text-muted flex-shrink-0">{disk.fsType}</span>
              </div>

              {/* 使用率进度条 */}
              <div className="mb-2">
                <div className="flex justify-between text-[10px] sm:text-xs mb-1">
                  <span className="text-muted">
                    {formatBytes(usedBytes)} / {formatBytes(disk.totalBytes)}
                  </span>
                  <span className={getUsageTextColor(disk.usagePct)}>
                    {disk.usagePct.toFixed(1)}%
                  </span>
                </div>
                <div className="h-1.5 sm:h-2 bg-[var(--background-secondary,#1f2937)] rounded-full overflow-hidden">
                  <div
                    className={`h-full rounded-full transition-all duration-300 ${getUsageColor(disk.usagePct)}`}
                    style={{ width: `${Math.min(100, disk.usagePct)}%` }}
                  />
                </div>
              </div>

              {/* I/O 详情 */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 text-[10px] sm:text-xs">
                <div>
                  <div className="text-muted">{nm.disk.read}</div>
                  <div className="text-default font-medium">{formatBytesPS(disk.readBytesPerSec)}</div>
                </div>
                <div>
                  <div className="text-muted">{nm.disk.write}</div>
                  <div className="text-default font-medium">{formatBytesPS(disk.writeBytesPerSec)}</div>
                </div>
                <div className="hidden sm:block">
                  <div className="text-muted">{nm.disk.iops}</div>
                  <div className="text-default font-medium">{(disk.readIOPS + disk.writeIOPS).toLocaleString()}</div>
                </div>
                <div className="hidden sm:block">
                  <div className="text-muted">{nm.disk.ioUtil}</div>
                  <div className={`font-medium ${getUsageTextColor(disk.ioUtilPct)}`}>
                    {disk.ioUtilPct.toFixed(1)}%
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* 底部汇总 */}
      <div className="mt-3 sm:mt-4 pt-3 sm:pt-4 border-t border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-muted" />
          <span className="text-[10px] sm:text-xs text-muted">{nm.disk.totalIops}</span>
        </div>
        <span className="text-xs sm:text-sm font-semibold text-default">{totalIOPS.toLocaleString()}</span>
      </div>
    </div>
  );
});
