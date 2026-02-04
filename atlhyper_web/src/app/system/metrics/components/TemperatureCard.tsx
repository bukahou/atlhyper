"use client";

import { memo } from "react";
import { Thermometer, AlertTriangle, Fan } from "lucide-react";
import type { TemperatureMetrics } from "@/types/node-metrics";

interface TemperatureCardProps {
  data: TemperatureMetrics;
}

const getTempColor = (temp: number, high?: number, critical?: number) => {
  if (critical && temp >= critical) return "text-red-500";
  if (high && temp >= high) return "text-yellow-500";
  if (temp >= 70) return "text-yellow-500";
  return "text-green-500";
};

const getTempBgColor = (temp: number, high?: number, critical?: number) => {
  if (critical && temp >= critical) return "bg-red-500";
  if (high && temp >= high) return "bg-yellow-500";
  if (temp >= 70) return "bg-yellow-500";
  return "bg-green-500";
};

export const TemperatureCard = memo(function TemperatureCard({ data }: TemperatureCardProps) {
  // 当没有有效温度数据时不触发警告
  const hasValidData = data.cpuTemp > 0 && data.cpuTempMax > 0;
  const isWarning = hasValidData && data.cpuTemp >= (data.cpuTempMax * 0.85);
  const isCritical = hasValidData && data.cpuTemp >= (data.cpuTempMax * 0.95);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-3 sm:mb-4">
        <div className="flex items-center gap-2">
          <div className={`p-1.5 sm:p-2 rounded-lg ${isCritical ? 'bg-red-500/10' : isWarning ? 'bg-yellow-500/10' : 'bg-cyan-500/10'}`}>
            <Thermometer className={`w-4 h-4 sm:w-5 sm:h-5 ${isCritical ? 'text-red-500' : isWarning ? 'text-yellow-500' : 'text-cyan-500'}`} />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">Temperature</h3>
            <p className="text-[10px] sm:text-xs text-muted">Max: {data.cpuTempMax > 0 ? `${data.cpuTempMax}°C` : 'N/A'}</p>
          </div>
        </div>
        {/* CPU 温度 */}
        <div className="text-right">
          <div className={`text-xl sm:text-2xl font-bold ${getTempColor(data.cpuTemp, data.cpuTempMax * 0.85, data.cpuTempMax)}`}>
            {data.cpuTemp.toFixed(1)}°C
          </div>
          <div className="text-[10px] sm:text-xs text-muted">CPU Temp</div>
        </div>
      </div>

      {/* 温度条 */}
      <div className="mb-3 sm:mb-4">
        <div className="relative h-3 sm:h-4 bg-[var(--background)] rounded-full overflow-hidden">
          {/* 温度刻度背景 */}
          <div className="absolute inset-0 flex">
            <div className="flex-1 bg-gradient-to-r from-blue-500/20 via-green-500/20 to-green-500/20" />
            <div className="w-[15%] bg-yellow-500/20" />
            <div className="w-[10%] bg-red-500/20" />
          </div>
          {/* 当前温度指示 */}
          <div
            className={`h-full rounded-full transition-all duration-300 ${getTempBgColor(data.cpuTemp, data.cpuTempMax * 0.85, data.cpuTempMax)}`}
            style={{ width: `${data.cpuTempMax > 0 ? Math.min(100, (data.cpuTemp / data.cpuTempMax) * 100) : 0}%`, opacity: 0.8 }}
          />
        </div>
        {/* 刻度 */}
        <div className="flex justify-between text-[10px] sm:text-xs text-muted mt-1">
          <span>0°C</span>
          <span>{Math.round(data.cpuTempMax * 0.5)}°C</span>
          <span>{data.cpuTempMax}°C</span>
        </div>
      </div>

      {/* GPU 温度 (如果有) */}
      {data.gpuTemp !== undefined && (
        <div className="mb-3 sm:mb-4 p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-1.5 sm:gap-2">
              <Fan className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-purple-500" />
              <span className="text-xs sm:text-sm text-default">GPU Temperature</span>
            </div>
            <span className={`text-base sm:text-lg font-bold ${getTempColor(data.gpuTemp, 75, 90)}`}>
              {data.gpuTemp.toFixed(1)}°C
            </span>
          </div>
        </div>
      )}

      {/* 传感器列表 */}
      <div>
        <div className="text-[10px] sm:text-xs text-muted mb-2">Sensors</div>
        <div className="space-y-1.5 sm:space-y-2 max-h-36 sm:max-h-48 overflow-y-auto">
          {data.sensors.map((sensor, index) => (
            <div key={index} className="flex items-center justify-between p-1.5 sm:p-2 bg-[var(--background)] rounded-lg">
              <div className="flex-1 min-w-0">
                <div className="text-xs sm:text-sm text-default truncate">{sensor.label}</div>
                <div className="text-[10px] sm:text-xs text-muted hidden sm:block">{sensor.name}</div>
              </div>
              <div className="flex items-center gap-1 sm:gap-2">
                <span className={`text-xs sm:text-sm font-medium ${getTempColor(sensor.temp, sensor.high, sensor.critical)}`}>
                  {sensor.temp.toFixed(1)}°C
                </span>
                {sensor.high && (
                  <span className="text-[10px] sm:text-xs text-muted hidden sm:inline">
                    (H:{sensor.high}°C)
                  </span>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* 告警提示 */}
      {(isWarning || isCritical) && (
        <div className={`mt-3 sm:mt-4 pt-3 sm:pt-4 border-t border-[var(--border-color)] flex items-center gap-2 ${isCritical ? 'text-red-500' : 'text-yellow-500'}`}>
          <AlertTriangle className="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" />
          <span className="text-xs sm:text-sm">
            {isCritical
              ? "Critical temperature!"
              : "High temperature warning"}
          </span>
        </div>
      )}
    </div>
  );
});
