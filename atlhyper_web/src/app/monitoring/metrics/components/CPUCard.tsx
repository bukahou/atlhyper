"use client";

import { memo } from "react";
import { Cpu, Gauge } from "lucide-react";
import type { CPUMetrics } from "@/types/node-metrics";
import { useI18n } from "@/i18n/context";

interface CPUCardProps {
  data: CPUMetrics;
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

export const CPUCard = memo(function CPUCard({ data }: CPUCardProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-3 sm:mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 sm:p-2 bg-orange-500/10 rounded-lg">
            <Cpu className="w-4 h-4 sm:w-5 sm:h-5 text-orange-500" />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">{nm.cpu.title}</h3>
            <p className="text-[10px] sm:text-xs text-muted">{data.coreCount}C/{data.threadCount}T @ {data.frequency.toFixed(0)} MHz</p>
          </div>
        </div>
        <div className="text-right">
          <div className={`text-xl sm:text-2xl font-bold ${getUsageTextColor(data.usagePercent)}`}>
            {data.usagePercent.toFixed(1)}%
          </div>
          <div className="text-[10px] sm:text-xs text-muted">{nm.cpu.usage}</div>
        </div>
      </div>

      {/* 核心使用率 */}
      {data.coreUsages && data.coreUsages.length > 0 && (
        <div className="mb-3 sm:mb-4 p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="flex items-center justify-between mb-2">
            <span className="text-[10px] sm:text-xs text-muted">{nm.cpu.coreUsage}</span>
            <span className="text-[10px] sm:text-xs text-muted">{data.coreUsages.length} {nm.cpu.threads}</span>
          </div>
          <div className="max-h-24 sm:max-h-32 overflow-y-auto [&::-webkit-scrollbar]:w-1.5 [&::-webkit-scrollbar-track]:bg-transparent [&::-webkit-scrollbar-thumb]:bg-[var(--border-color)] [&::-webkit-scrollbar-thumb]:rounded-full hover:[&::-webkit-scrollbar-thumb]:bg-muted/50">
            <div className="grid grid-cols-4 sm:grid-cols-5 gap-1.5 sm:gap-2 pr-1">
              {data.coreUsages.map((usage, i) => (
                <div key={i} className="flex flex-col items-center">
                  <div className="w-full h-8 sm:h-10 bg-[var(--border-color)] rounded-sm overflow-hidden flex flex-col-reverse">
                    <div
                      className={`w-full transition-all ${getUsageColor(usage)}`}
                      style={{ height: `${Math.max(usage, 2)}%` }}
                    />
                  </div>
                  <span className="text-[8px] sm:text-[10px] text-muted mt-0.5 sm:mt-1">{i}</span>
                </div>
              ))}
            </div>
          </div>
        </div>
      )}

      {/* CPU 型号 */}
      <div className="mb-3 sm:mb-4 p-2 bg-[var(--background)] rounded-lg">
        <p className="text-[10px] sm:text-xs text-muted truncate" title={data.model}>
          {data.model}
        </p>
      </div>

      {/* 负载平均值 - 底部 */}
      <div className="flex flex-wrap items-center gap-1.5 sm:gap-2 pt-2 sm:pt-3 border-t border-[var(--border-color)]">
        <Gauge className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-muted" />
        <span className="text-[10px] sm:text-xs text-muted">{nm.cpu.load}:</span>
        <div className="flex gap-2 sm:gap-3 text-xs sm:text-sm">
          <span className="font-medium text-default">
            <span className="text-muted">1m:</span> {data.loadAvg1.toFixed(2)}
          </span>
          <span className="font-medium text-default">
            <span className="text-muted">5m:</span> {data.loadAvg5.toFixed(2)}
          </span>
          <span className="font-medium text-default hidden sm:inline">
            <span className="text-muted">15m:</span> {data.loadAvg15.toFixed(2)}
          </span>
        </div>
      </div>
    </div>
  );
});
