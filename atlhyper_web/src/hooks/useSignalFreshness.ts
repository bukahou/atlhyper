/**
 * 当前集群某个信号的新鲜度
 *
 * 四个观测页共用：空数据时用它说清楚是「没有流量」还是「采集异常」。
 * 判定在 Master 完成；取数是共用的一份查询（同一集群只请求一次），这里只挑出本页关心的那个信号。
 */

"use client";

import { useQuery } from "@tanstack/react-query";
import { signalFreshnessQuery } from "@/queries/observe-freshness";
import type { FreshnessResponse, SignalFreshnessItem } from "@/types/observe";
import { useCurrentCluster } from "./useCurrentCluster";

/** 从接口返回的全部信号里挑出一个；没有就是 null（徽章显示「未知」） */
export function pickSignalFreshness(
  freshness: FreshnessResponse | null | undefined,
  signal: SignalFreshnessItem["signal"],
): SignalFreshnessItem | null {
  return freshness?.signals?.find((item) => item.signal === signal) ?? null;
}

export function useSignalFreshness(signal: SignalFreshnessItem["signal"]): SignalFreshnessItem | null {
  const { clusterId } = useCurrentCluster();
  const query = useQuery(signalFreshnessQuery(clusterId));
  // 最近一次取数失败就显示「未知」，不继续显示上一次的结果：
  // 新鲜度说的是「现在」，拿不到就是不知道。它是辅助信息，失败不该让页面报错
  if (query.isError) return null;
  return pickSignalFreshness(query.data, signal);
}
