/**
 * QueryView 显示哪一种状态的判定（纯函数，便于单测）
 */

export type ViewState = "no-cluster" | "loading" | "error" | "stale" | "empty" | "ready";

export interface ViewStateInput {
  /** 这块内容是否按集群取数 */
  requireCluster: boolean;
  /** 集群列表是否已加载完（无论成功失败） */
  clusterListLoaded: boolean;
  /** 是否已选定集群 */
  hasCluster: boolean;
  /** 是否已有数据（包括上一次成功的结果） */
  hasData: boolean;
  /** 数据是否为空 */
  isEmpty: boolean;
  /** 还没有数据，且尚未得到结果 */
  isPending: boolean;
  isError: boolean;
  /** 已有数据的情况下，后台刷新失败 */
  isRefetchError: boolean;
}

/**
 * 判定顺序固定：无集群 → 加载中 → 出错 → 刷新失败 → 空 → 就绪。
 *
 * 「无集群」必须先于「加载中」：没有集群时查询不会启用、一直处于 pending，
 * 顺序反了页面就会永远转圈。集群列表还没加载完时不能断定「无集群」，按加载中处理。
 *
 * 已经有数据时不显示「无集群」：演示数据不依赖集群，没有集群也照常显示。
 */
export function resolveViewState(input: ViewStateInput): ViewState {
  if (!input.hasData) {
    if (input.requireCluster && !input.hasCluster) {
      return input.clusterListLoaded ? "no-cluster" : "loading";
    }
    if (input.isError) return "error";
    return "loading";
  }
  if (input.isRefetchError) return "stale";
  if (input.isEmpty) return "empty";
  return "ready";
}
