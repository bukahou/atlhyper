/**
 * AIOps 风险分数工具（单一信任源）
 *
 * 单位契约（与后端 atlhyper_master_v2/gateway/handler/aiops/scale_risk.go 对齐）：
 *   - AIOps 核心层（Go aiops/risk/）: [0, 1] 浮点概率
 *   - Gateway/API 层: [0, 100] 整数（已 Scale）
 *   - 前端 API 类型以下全部: [0, 100] 整数
 *
 * 前端禁止再对 rLocal / rWeighted / rFinal 字段做 × 100 或 ÷ 100，
 * 禁止再写 `>= 0.8` 这类按 [0,1] 假设的阈值判断。
 * 一律通过本模块导出的函数消费分数。
 *
 * 历史背景：曾出现拓扑图 badge 显示 4500（实际 45%）和节点全红的 bug
 * （2026-04，commit 712f902 之后的调查），根因即分散在多处的 × 100 和
 * 旧阈值残留。本模块即为根治手段。
 */

/** 百分制阈值（递减排列） */
export const RISK_THRESHOLDS = {
  /** 严重：大于等于此分数显示红色 */
  critical: 80,
  /** 警告：大于等于此分数显示橙色 */
  warning: 50,
  /** 注意：大于等于此分数显示黄色 */
  caution: 30,
  /** 信息：大于等于此分数显示蓝色（视为"有风险"起点） */
  info: 10,
} as const;

/** 风险等级离散化 */
export type RiskLevel = "critical" | "warning" | "caution" | "info" | "healthy";

/** 每个等级对应的显示颜色（hex） */
const LEVEL_COLOR: Record<RiskLevel, string> = {
  critical: "#ef4444",
  warning: "#f97316",
  caution: "#eab308",
  info: "#3b82f6",
  healthy: "#22c55e",
};

/**
 * 百分制分数 → 风险等级
 * @param score 百分制 [0, 100]
 */
export function riskLevel(score: number): RiskLevel {
  if (score >= RISK_THRESHOLDS.critical) return "critical";
  if (score >= RISK_THRESHOLDS.warning) return "warning";
  if (score >= RISK_THRESHOLDS.caution) return "caution";
  if (score >= RISK_THRESHOLDS.info) return "info";
  return "healthy";
}

/**
 * 百分制分数 → hex 颜色（拓扑染色、badge 背景、文字色共用）
 * @param score 百分制 [0, 100]
 */
export function riskColor(score: number): string {
  return LEVEL_COLOR[riskLevel(score)];
}

/**
 * 百分制分数 → 显示字符串（整数，无单位）
 * @param score 百分制 [0, 100]
 * @returns 例如 45 → "45"
 */
export function formatRiskScore(score: number): string {
  return Math.round(score).toString();
}

/**
 * 分数是否构成风险（>= info 阈值 10）
 * @param score 百分制 [0, 100]
 */
export function isRisky(score: number): boolean {
  return score >= RISK_THRESHOLDS.info;
}
