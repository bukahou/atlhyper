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

/**
 * 上一份查询与这一次是不是只差页码（筛选条件相同）。
 *
 * 翻页时先留着上一页，界面不闪；但换了筛选条件就不留 ——
 * 否则「已结束」的筛选下会先显示一屏「持续中」的告警。
 * 约定见 keys.ts：键的第三段是范围，分页用 limit / offset。
 */
export function isSameFilterOtherPage<TFilter extends { limit: number; offset: number }>(
  query: { queryKey: readonly unknown[] } | undefined,
  filter: TFilter,
): boolean {
  const previousScope = query?.queryKey[2];
  if (typeof previousScope !== "object" || previousScope === null) return false;
  const withoutPage = (scope: object) =>
    Object.entries(scope)
      .filter(([field]) => field !== "limit" && field !== "offset")
      .sort(([fieldA], [fieldB]) => fieldA.localeCompare(fieldB));
  return JSON.stringify(withoutPage(previousScope)) === JSON.stringify(withoutPage(filter));
}
