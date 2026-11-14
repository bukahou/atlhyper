/**
 * 换筛选条件时要不要先留着上一份结果（placeholderData）
 */

/**
 * 上一份查询是不是同一个集群的。
 *
 * 换筛选条件、翻页时先留着上一份结果，界面不闪；但只在同一个集群内 ——
 * 换了集群还显示旧集群的数据，等于把别的集群的东西当成当前集群的。
 * 约定见 keys.ts：键的第三段是范围，里面带 clusterId。
 */
export function isQueryOfCluster(query: { queryKey: readonly unknown[] } | undefined, clusterId: string): boolean {
  const scope = query?.queryKey[2] as { clusterId?: string } | undefined;
  return scope?.clusterId === clusterId;
}
