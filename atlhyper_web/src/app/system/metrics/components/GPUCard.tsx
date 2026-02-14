"use client";

import { memo } from "react";
import { Zap, Thermometer, Fan, Cpu } from "lucide-react";
import type { GPUMetrics } from "@/types/node-metrics";
import { formatBytes } from "@/lib/format";

interface GPUCardProps {
  data: GPUMetrics[];
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

export const GPUCard = memo(function GPUCard({ data }: GPUCardProps) {
  if (!data || data.length === 0) {
    return null;
  }

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      {/* 头部 */}
      <div className="flex items-center gap-2 mb-4">
        <div className="p-2 bg-emerald-500/10 rounded-lg">
          <Zap className="w-5 h-5 text-emerald-500" />
        </div>
        <div>
          <h3 className="text-base font-semibold text-default">GPU</h3>
          <p className="text-xs text-muted">{data.length} device(s)</p>
        </div>
      </div>

      {/* GPU 列表 */}
      <div className="space-y-4">
        {data.map((gpu) => (
          <div key={gpu.index} className="p-4 bg-[var(--background)] rounded-lg">
            {/* GPU 名称 */}
            <div className="flex items-center justify-between mb-3">
              <div>
                <div className="text-sm font-medium text-default">{gpu.name}</div>
                <div className="text-xs text-muted">GPU #{gpu.index}</div>
              </div>
              <div className="text-right">
                <div className={`text-xl font-bold ${getUsageTextColor(gpu.gpuUtilization)}`}>
                  {gpu.gpuUtilization}%
                </div>
                <div className="text-xs text-muted">Utilization</div>
              </div>
            </div>

            {/* GPU 使用率进度条 */}
            <div className="mb-4">
              <div className="flex justify-between text-xs mb-1">
                <span className="text-muted">GPU Compute</span>
                <span className="text-default">{gpu.gpuUtilization}%</span>
              </div>
              <div className="h-2 bg-[var(--background-secondary,#1f2937)] rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all duration-300 ${getUsageColor(gpu.gpuUtilization)}`}
                  style={{ width: `${Math.min(100, gpu.gpuUtilization)}%` }}
                />
              </div>
            </div>

            {/* 显存 */}
            <div className="mb-4">
              <div className="flex justify-between text-xs mb-1">
                <span className="text-muted">Memory</span>
                <span className="text-default">
                  {formatBytes(gpu.memoryUsed)} / {formatBytes(gpu.memoryTotal)}
                </span>
              </div>
              <div className="h-2 bg-[var(--background-secondary,#1f2937)] rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full transition-all duration-300 ${getUsageColor(gpu.memUtilization)}`}
                  style={{ width: `${Math.min(100, gpu.memUtilization)}%` }}
                />
              </div>
            </div>

            {/* 指标卡片 */}
            <div className="grid grid-cols-4 gap-2">
              <div className="p-2 bg-[var(--background-secondary,#1f2937)] rounded-lg text-center">
                <Thermometer className="w-4 h-4 mx-auto mb-1 text-red-500" />
                <div className={`text-sm font-medium ${gpu.temperature >= 80 ? "text-red-500" : "text-default"}`}>
                  {gpu.temperature}°C
                </div>
                <div className="text-xs text-muted">Temp</div>
              </div>
              <div className="p-2 bg-[var(--background-secondary,#1f2937)] rounded-lg text-center">
                <Fan className="w-4 h-4 mx-auto mb-1 text-blue-500" />
                <div className="text-sm font-medium text-default">{gpu.fanSpeed}%</div>
                <div className="text-xs text-muted">Fan</div>
              </div>
              <div className="p-2 bg-[var(--background-secondary,#1f2937)] rounded-lg text-center">
                <Zap className="w-4 h-4 mx-auto mb-1 text-yellow-500" />
                <div className="text-sm font-medium text-default">{gpu.powerUsage}W</div>
                <div className="text-xs text-muted">Power</div>
              </div>
              <div className="p-2 bg-[var(--background-secondary,#1f2937)] rounded-lg text-center">
                <Cpu className="w-4 h-4 mx-auto mb-1 text-purple-500" />
                <div className="text-sm font-medium text-default">{gpu.memUtilization}%</div>
                <div className="text-xs text-muted">VRAM</div>
              </div>
            </div>

            {/* GPU 进程 */}
            {gpu.processes.length > 0 && (
              <div className="mt-4 pt-4 border-t border-[var(--border-color)]">
                <div className="text-xs text-muted mb-2">GPU Processes</div>
                <div className="space-y-1">
                  {gpu.processes.map((proc) => (
                    <div
                      key={proc.pid}
                      className="flex items-center justify-between text-xs py-1"
                    >
                      <div className="flex items-center gap-2">
                        <span className="text-muted">{proc.pid}</span>
                        <span className="text-default">{proc.name}</span>
                      </div>
                      <span className="text-muted">{formatBytes(proc.memoryUsed)}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
});
