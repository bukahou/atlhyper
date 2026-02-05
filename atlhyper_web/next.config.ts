import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    unoptimized: true,
  },
  // API 代理通过 middleware.ts 实现（运行时读取 API_URL 环境变量）
};

export default nextConfig;
