/**
 * 一份按集群取数的数据现在能不能取
 */
import type { DataSourceMode } from "@/config/data-source";

/**
 * 真实接口需要先选定集群；演示数据不依赖集群。
 *
 * 只部署了前端做演示、或还没有任何集群接入时，演示模式仍然要能显示演示数据。
 */
export function clusterDataAvailable(clusterId: string, mode: DataSourceMode): boolean {
  return mode === "mock" || clusterId !== "";
}
