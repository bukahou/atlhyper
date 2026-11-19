/**
 * AIOps 风险等级的显示（单一来源）。
 *
 * 等级由后端判定并随数据下发（实体：healthy / low / medium / high / critical，见 aiops.RiskLevel；
 * 集群：healthy / low / warning / critical，见 aiops.ClusterRiskLevel）。
 * 前端 ⛔ 不按分数再判一次等级（此前前端另有一套阈值，拓扑图的颜色与同一实体的等级徽章对不上），
 * 只负责「等级 → 颜色」。分数（[0, 100]，网关层已换算）只用于显示数字。
 */

/** 后端下发的风险等级 */
export type RiskLevel = "healthy" | "low" | "medium" | "warning" | "high" | "critical";

/** 等级 → 颜色（hex，拓扑染色、徽章背景共用） */
const LEVEL_COLOR: Record<RiskLevel, string> = {
  critical: "#ef4444",
  high: "#f97316",
  medium: "#eab308",
  warning: "#eab308",
  low: "#3b82f6",
  healthy: "#22c55e",
};

/** 后端等级 → 颜色；没有等级或不认识的等级按健康显示 */
export function riskLevelColor(level: string | undefined): string {
  return LEVEL_COLOR[(level ?? "healthy") as RiskLevel] ?? LEVEL_COLOR.healthy;
}

/** 等级 → 徽章样式（背景 + 文字）。风险、AIOps 事件严重度、通知严重度是同一套等级，共用这一份 */
const LEVEL_BADGE: Record<RiskLevel, string> = {
  critical: "bg-red-500/15 text-red-600 dark:text-red-400",
  high: "bg-orange-500/15 text-orange-600 dark:text-orange-400",
  medium: "bg-yellow-500/15 text-yellow-700 dark:text-yellow-400",
  warning: "bg-yellow-500/15 text-yellow-700 dark:text-yellow-400",
  low: "bg-blue-500/15 text-blue-600 dark:text-blue-400",
  healthy: "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400",
};

/** 不认识的等级：中性灰，既不显示成正常也不显示成故障 */
const UNKNOWN_BADGE = "bg-gray-500/15 text-muted";

export function levelBadgeClass(level: string | undefined): string {
  return LEVEL_BADGE[level as RiskLevel] ?? UNKNOWN_BADGE;
}

/** 是否属于需要突出显示的高风险（high / critical） */
export function isElevatedRisk(level: string | undefined): boolean {
  return level === "high" || level === "critical";
}

/**
 * 百分制分数 → 显示字符串（整数，无单位）
 * @param score 百分制 [0, 100]
 * @returns 例如 45 → "45"
 */
export function formatRiskScore(score: number): string {
  return Math.round(score).toString();
}
