"use client";

import type { StatusLevel } from "@/types/cluster";

interface StatusBadgeProps {
  status: string;
  /** 显示等级：资源状态由后端判定后随数据下发；布尔开关等由调用方按语义给出。⛔ 不按文字猜 */
  type: StatusLevel;
}

const statusColors: Record<StatusLevel, string> = {
  success: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  warning: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  error: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  info: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  default: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300",
};

export function StatusBadge({ status, type }: StatusBadgeProps) {
  return (
    <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${statusColors[type] ?? statusColors.default}`}>
      {status}
    </span>
  );
}
