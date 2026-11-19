// 硬件健康状态 → 颜色（只有这一份）。判定全部来自后端，这里只负责上色。
// unknown = 没有这个读数：灰色，既不显示成正常也不显示成故障
import type { HardwareStatus } from "@/types/hardware";

/** 状态 → 文字色 */
export function hardwareStatusText(s: HardwareStatus): string {
  switch (s) {
    case "crit":
      return "text-red-500";
    case "warn":
      return "text-yellow-500";
    case "good":
      return "text-green-500";
    default:
      return "text-muted";
  }
}

/** 状态 → chip（背景 + 文字） */
export function hardwareStatusChip(s: HardwareStatus): string {
  switch (s) {
    case "crit":
      return "bg-red-500/10 text-red-500";
    case "warn":
      return "bg-yellow-500/10 text-yellow-500";
    case "good":
      return "bg-green-500/10 text-green-500";
    default:
      return "bg-gray-500/10 text-muted";
  }
}

/** 状态 → 进度条填充色 */
export function hardwareStatusBar(s: HardwareStatus): string {
  switch (s) {
    case "crit":
      return "bg-red-500";
    case "warn":
      return "bg-yellow-500";
    case "good":
      return "bg-green-500";
    default:
      return "bg-gray-400";
  }
}
