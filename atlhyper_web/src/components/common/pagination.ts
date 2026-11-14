/**
 * 分页的页码计算
 */

/**
 * 把页码收拢到有效范围 [1, totalPages]。
 *
 * 没有数据（totalPages 为 0）时仍返回 1：页码从 1 起算，不存在第 0 页。
 */
export function clampPage(requestedPage: number, totalPages: number): number {
  return Math.min(Math.max(requestedPage, 1), Math.max(totalPages, 1));
}
