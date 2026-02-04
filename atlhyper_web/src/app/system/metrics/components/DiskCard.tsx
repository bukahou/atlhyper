"use client";

import { memo } from "react";
import { Database, ArrowDown, ArrowUp, Activity } from "lucide-react";
import type { DiskMetrics } from "@/types/node-metrics";
import { formatBytes, formatBytesPS } from "../mock/data";

interface DiskCardProps {
  data: DiskMetrics[];
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
  // 计算总 I/O
  const totalReadPS = data.reduce((acc, d) => acc + d.readBytesPS, 0);
  const totalWritePS = data.reduce((acc, d) => acc + d.writeBytesPS, 0);
  const totalIOPS = data.reduce((acc, d) => acc + d.iops, 0);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-purple-500/10 rounded-lg">
            <Database className="w-5 h-5 text-purple-500" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-default">Disk</h3>
            <p className="text-xs text-muted">{data.length} mount(s)</p>
          </div>
        </div>
        {/* 总 I/O 速率 */}
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1 text-sm">
            <ArrowDown className="w-3 h-3 text-blue-500" />
            <span className="text-default">{formatBytesPS(totalReadPS)}</span>
          </div>
          <div className="flex items-center gap-1 text-sm">
            <ArrowUp className="w-3 h-3 text-orange-500" />
            <span className="text-default">{formatBytesPS(totalWritePS)}</span>
          </div>
        </div>
      </div>

      {/* 磁盘列表 */}
      <div className="space-y-4">
        {data.map((disk) => (
          <div key={disk.mountPoint} className="p-3 bg-[var(--background)] rounded-lg">
            {/* 设备 & 挂载点 */}
            <div className="flex items-center justify-between mb-2">
              <div>
                <span className="text-sm font-medium text-default">{disk.mountPoint}</span>
                <span className="text-xs text-muted ml-2">({disk.device})</span>
              </div>
              <span className="text-xs text-muted">{disk.fsType}</span>
            </div>

            {/* 使用率进度条 */}
            <div className="mb-2">
              <div className="flex justify-between text-xs mb-1">
                <span className="text-muted">
                  {formatBytes(disk.usedBytes)} / {formatBytes(disk.totalBytes)}
                </span>
                <span className={getUsageTextColor(disk.usagePercent)}>
                  {disk.usagePercent.toFixed(1)}%
                </span>
              </div>
              <div className="h-2 bg-[var(--background-secondary,#1f2937)] rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all duration-300 ${getUsageColor(disk.usagePercent)}`}
                  style={{ width: `${Math.min(100, disk.usagePercent)}%` }}
                />
              </div>
            </div>

            {/* I/O 详情 */}
            <div className="grid grid-cols-4 gap-2 text-xs">
              <div>
                <div className="text-muted">Read</div>
                <div className="text-default font-medium">{formatBytesPS(disk.readBytesPS)}</div>
              </div>
              <div>
                <div className="text-muted">Write</div>
                <div className="text-default font-medium">{formatBytesPS(disk.writeBytesPS)}</div>
              </div>
              <div>
                <div className="text-muted">IOPS</div>
                <div className="text-default font-medium">{disk.iops.toLocaleString()}</div>
              </div>
              <div>
                <div className="text-muted">IO Util</div>
                <div className={`font-medium ${getUsageTextColor(disk.ioUtil)}`}>
                  {disk.ioUtil.toFixed(1)}%
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* 底部汇总 */}
      <div className="mt-4 pt-4 border-t border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Activity className="w-4 h-4 text-muted" />
          <span className="text-xs text-muted">Total IOPS</span>
        </div>
        <span className="text-sm font-semibold text-default">{totalIOPS.toLocaleString()}</span>
      </div>
    </div>
  );
});
