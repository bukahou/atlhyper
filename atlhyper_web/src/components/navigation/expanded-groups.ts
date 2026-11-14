/**
 * 侧栏分组的展开状态
 */

/**
 * 在已展开的分组之上，把 groupsToExpand 也展开；已经展开的不重复，其余分组不动。
 *
 * 没有新增时返回原数组（引用不变），调用方据此可以跳过一次无意义的状态更新。
 */
export function withGroupsExpanded(expandedGroups: string[], groupsToExpand: string[]): string[] {
  const missingGroups = groupsToExpand.filter((group) => !expandedGroups.includes(group));
  return missingGroups.length === 0 ? expandedGroups : [...expandedGroups, ...missingGroups];
}
