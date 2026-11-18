import type { Point } from "@/types/node-metrics";

/** 一张趋势图的显示状态：画图 / 加载失败 / 暂无数据 */
export type HistoryChartState = "data" | "failed" | "empty";

/**
 * 一张趋势图（一个指标、若干节点）该显示什么。
 * 任何一个节点有这个指标的数据就画图；都没有数据时，只要名单里有节点的请求失败就显示「加载失败」，
 * 否则显示「暂无数据」—— 失败和「确实没数据」必须分开，不然排查时会被误导。
 */
export function historyChartState(
  nodeNames: readonly string[],
  metricKey: string,
  historyMap: Record<string, Record<string, Point[]>>,
  failedNodes: ReadonlySet<string>,
): HistoryChartState {
  if (nodeNames.some((name) => (historyMap[name]?.[metricKey]?.length ?? 0) > 0)) return "data";
  return nodeNames.some((name) => failedNodes.has(name)) ? "failed" : "empty";
}
