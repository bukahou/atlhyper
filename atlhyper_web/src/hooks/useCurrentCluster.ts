import { useClusterStore } from "@/store/clusterStore";

export interface CurrentCluster {
  /** 当前选定的集群；没有选定时为空串 */
  clusterId: string;
  hasCluster: boolean;
  /** 集群列表是否已加载完。为假时不能断定「没有集群」 */
  listLoaded: boolean;
}

/** 当前集群。按集群取数的查询把 clusterId 传给 queries/ 下的工厂 */
export function useCurrentCluster(): CurrentCluster {
  const clusterId = useClusterStore((state) => state.currentClusterId);
  const listLoaded = useClusterStore((state) => state.listLoaded);
  return { clusterId, hasCluster: clusterId !== "", listLoaded };
}
