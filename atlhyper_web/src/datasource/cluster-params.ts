/**
 * Cluster 数据源各文件共用的参数形态（与 api 层保持一致的字段名）
 */

/** 列表查询：按集群取，namespace 可选（不传 = 全部 namespace） */
export interface NamespacedListParams {
  cluster_id: string;
  namespace?: string;
}

/** 详情查询：带 namespace 的资源里的某一个对象 */
export interface NamespacedResourceParams {
  ClusterID: string;
  Namespace: string;
  Name: string;
}
