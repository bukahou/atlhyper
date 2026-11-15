import { MOCK_NOTIFY_CLUSTER_IDS } from "@/mock/notify";
import { useClusterStore } from "@/store/clusterStore";

/**
 * 规则与静默表单里可选的集群。
 *
 * 真实模式用当前已接入的集群；演示模式用演示集群 —— 演示视图里不混入真实的集群名。
 */
export function useNotifyClusterIds(isDemo: boolean): string[] {
  const connectedClusterIds = useClusterStore((state) => state.clusterIds);
  return isDemo ? MOCK_NOTIFY_CLUSTER_IDS : connectedClusterIds;
}
