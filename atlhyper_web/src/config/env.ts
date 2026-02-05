/**
 * 环境变量配置
 * 集中管理所有环境变量，提供类型安全
 */

const isDev = process.env.NODE_ENV === "development";

export const env = {
  // API 配置
  // 统一使用相对路径，通过 Next.js rewrites 代理到后端
  // 后端地址在 next.config.ts 中通过环境变量配置（运行时读取）
  apiUrl: "",

  // 数据刷新间隔（毫秒），默认 30 秒
  refreshInterval: Number(process.env.NEXT_PUBLIC_REFRESH_INTERVAL) || 30000,

  // 运行环境
  isDev,
  isProd: process.env.NODE_ENV === "production",
} as const;

// 导出类型
export type Env = typeof env;
