/**
 * 集群配置
 * 使用 clusterStore 管理多集群状态
 */

// 默认集群 ID（与后端 Agent 配置一致）
export const DEFAULT_CLUSTER_ID = "ZGMF-X10A";

/**
 * 获取当前使用的集群 ID
 * 优先使用 localStorage 中保存的 currentClusterId，否则使用默认值
 *
 * 注意：这个函数在服务端渲染时返回默认值
 * 在客户端，clusterStore 会管理实际的集群选择
 */
export function getCurrentClusterId(): string {
  if (typeof window === "undefined") {
    return DEFAULT_CLUSTER_ID;
  }

  // 优先使用用户选择的当前集群
  const currentCluster = localStorage.getItem("currentClusterId");
  if (currentCluster) {
    return currentCluster;
  }

  // 其次使用登录后获取的集群列表的第一个
  const clusterIdsStr = localStorage.getItem("clusterIds");
  if (clusterIdsStr) {
    try {
      const clusterIds = JSON.parse(clusterIdsStr);
      if (Array.isArray(clusterIds) && clusterIds.length > 0) {
        return clusterIds[0];
      }
    } catch {
      // 解析失败，使用默认值
    }
  }

  return DEFAULT_CLUSTER_ID;
}
