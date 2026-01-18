import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",
  images: {
    unoptimized: true,
  },
  // 开发环境代理配置，解决 CORS 跨域问题
  async rewrites() {
    const apiBase = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
    return [
      // Master V2 API
      {
        source: "/api/v2/:path*",
        destination: `${apiBase}/api/v2/:path*`,
      },
      // 旧 API（兼容）
      {
        source: "/uiapi/:path*",
        destination: `${apiBase}/uiapi/:path*`,
      },
      {
        source: "/ingest/:path*",
        destination: `${apiBase}/ingest/:path*`,
      },
      {
        source: "/ai/:path*",
        destination: `${apiBase}/ai/:path*`,
      },
    ];
  },
};

export default nextConfig;
