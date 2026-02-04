"use client";

import { memo } from "react";
import { HardDrive, RefreshCw } from "lucide-react";
import type { MemoryMetrics } from "@/types/node-metrics";
import { formatBytes } from "../mock/data";

interface MemoryCardProps {
  data: MemoryMetrics;
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

export const MemoryCard = memo(function MemoryCard({ data }: MemoryCardProps) {
  const usedPercent = data.usagePercent;
  const cachedPercent = (data.cached / data.totalBytes) * 100;
  const buffersPercent = (data.buffers / data.totalBytes) * 100;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-green-500/10 rounded-lg">
            <HardDrive className="w-5 h-5 text-green-500" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-default">Memory</h3>
            <p className="text-xs text-muted">Total: {formatBytes(data.totalBytes)}</p>
          </div>
        </div>
        <div className="text-right">
          <div className={`text-2xl font-bold ${getUsageTextColor(usedPercent)}`}>
            {usedPercent.toFixed(1)}%
          </div>
          <div className="text-xs text-muted">Usage</div>
        </div>
      </div>

      {/* 内存使用条 (分段显示) */}
      <div className="mb-4">
        <div className="h-4 bg-[var(--background)] rounded-full overflow-hidden flex">
          {/* Used */}
          <div
            className="h-full bg-green-500 transition-all duration-300"
            style={{ width: `${usedPercent - cachedPercent - buffersPercent}%` }}
            title={`Used: ${formatBytes(data.usedBytes - data.cached - data.buffers)}`}
          />
          {/* Cached */}
          <div
            className="h-full bg-blue-500 transition-all duration-300"
            style={{ width: `${cachedPercent}%` }}
            title={`Cached: ${formatBytes(data.cached)}`}
          />
          {/* Buffers */}
          <div
            className="h-full bg-purple-500 transition-all duration-300"
            style={{ width: `${buffersPercent}%` }}
            title={`Buffers: ${formatBytes(data.buffers)}`}
          />
        </div>
        {/* 图例 */}
        <div className="flex items-center gap-4 mt-2 text-xs">
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-green-500" />
            <span className="text-muted">Used</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-blue-500" />
            <span className="text-muted">Cached</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-purple-500" />
            <span className="text-muted">Buffers</span>
          </div>
        </div>
      </div>

      {/* 详细数据 */}
      <div className="grid grid-cols-2 gap-3 mb-4">
        <div className="p-3 bg-[var(--background)] rounded-lg">
          <div className="text-xs text-muted mb-1">Used</div>
          <div className="text-sm font-semibold text-default">{formatBytes(data.usedBytes)}</div>
        </div>
        <div className="p-3 bg-[var(--background)] rounded-lg">
          <div className="text-xs text-muted mb-1">Available</div>
          <div className="text-sm font-semibold text-default">{formatBytes(data.availableBytes)}</div>
        </div>
        <div className="p-3 bg-[var(--background)] rounded-lg">
          <div className="text-xs text-muted mb-1">Cached</div>
          <div className="text-sm font-semibold text-blue-500">{formatBytes(data.cached)}</div>
        </div>
        <div className="p-3 bg-[var(--background)] rounded-lg">
          <div className="text-xs text-muted mb-1">Buffers</div>
          <div className="text-sm font-semibold text-purple-500">{formatBytes(data.buffers)}</div>
        </div>
      </div>

      {/* Swap */}
      {data.swapTotalBytes > 0 && (
        <div className="pt-4 border-t border-[var(--border-color)]">
          <div className="flex items-center gap-2 mb-2">
            <RefreshCw className="w-4 h-4 text-muted" />
            <span className="text-sm font-medium text-default">Swap</span>
            <span className="text-xs text-muted ml-auto">
              {formatBytes(data.swapUsedBytes)} / {formatBytes(data.swapTotalBytes)}
            </span>
          </div>
          <div className="h-2 bg-[var(--background)] rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all duration-300 ${getUsageColor(data.swapUsagePercent)}`}
              style={{ width: `${Math.min(100, data.swapUsagePercent)}%` }}
            />
          </div>
          <div className="text-xs text-muted mt-1 text-right">
            {data.swapUsagePercent.toFixed(1)}% used
          </div>
        </div>
      )}
    </div>
  );
});
