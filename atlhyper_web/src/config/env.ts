/**
 * 环境变量配置
 * 集中管理所有环境变量，提供类型安全
 */

const isDev = process.env.NODE_ENV === "development";

export const env = {
  // API 配置
  // 开发环境使用相对路径 (走 Next.js 代理)
  // 生产环境使用完整 URL
  apiUrl: isDev ? "" : (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"),

  // 数据刷新间隔（毫秒），默认 30 秒
  refreshInterval: Number(process.env.NEXT_PUBLIC_REFRESH_INTERVAL) || 30000,

  // 运行环境
  isDev,
  isProd: process.env.NODE_ENV === "production",
} as const;

// 导出类型
export type Env = typeof env;
