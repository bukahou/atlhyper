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
  /** 要跟得上、但每次查询都不轻的数据（日志列表与直方图） */
  moderate: 15_000,
  /** 变化慢或查询重的数据 */
  slow: 30_000,
  /** 以分钟为粒度变化的数据（趋势图的历史点、信号新鲜度）：刷得更勤也看不到新东西 */
  minute: 60_000,
} as const;
