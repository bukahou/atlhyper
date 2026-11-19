import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    unoptimized: true,
  },
  // API 代理通过 middleware.ts 实现（运行时读取 API_URL 环境变量）
  experimental: {
    // 代理到后端的请求在这么久没有数据时断开（默认 30 秒）。经指令通道的查询最多等 Agent 30 秒，
    // 后端普通接口的写超时是 60 秒；AI 对话的流每 15 秒有心跳，不受它影响
    proxyTimeout: 120_000,
  },
};

export default nextConfig;
