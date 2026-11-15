/** 通知设置页各表格共用的显示格式 */

/** RFC 3339 时间 → 本地时间；空串（「没有」）显示为 "-" */
export function formatTimestamp(isoTimestamp: string): string {
  return isoTimestamp ? new Date(isoTimestamp).toLocaleString() : "-";
}

/** 这个页签收到的通用属性 */
export interface NotifyTabProps {
  /** 没有 notify.manage：整个页签显示演示数据，写操作不可用 */
  isDemo: boolean;
}

/**
 * 从一份完整的列表里切出一页，形状与后端分页接口的结果一致。
 *
 * 只给演示数据用：演示数据是前端手里的一份完整列表，真实数据由后端分页。
 */
export function demoPage<T>(items: T[], page: number, pageSize: number): { items: T[]; total: number } {
  return { items: items.slice(page * pageSize, (page + 1) * pageSize), total: items.length };
}
