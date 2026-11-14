/**
 * 轮询间隔（毫秒）
 *
 * 页面不自己写数字，从这里选一档。
 */
import { env } from "@/config/env";

export const REFRESH = {
  /** 变化快、需要接近实时的数据 */
  fast: 10_000,
  /** 默认档，由 NEXT_PUBLIC_REFRESH_INTERVAL 配置 */
  standard: env.refreshInterval,
  /** 变化慢或查询重的数据 */
  slow: 30_000,
} as const;
