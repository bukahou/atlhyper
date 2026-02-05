import { NextRequest, NextResponse } from "next/server";

/**
 * API 代理中间件
 * 将 /api/v2/* 等请求代理到后端 Controller
 * API_URL 环境变量在运行时读取（类似 Kibana）
 */
export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // 需要代理的路径前缀
  const proxyPaths = ["/api/v2", "/uiapi", "/ingest", "/ai"];

  // 检查是否需要代理
  const shouldProxy = proxyPaths.some((prefix) => pathname.startsWith(prefix));

  if (!shouldProxy) {
    return NextResponse.next();
  }

  // 从环境变量读取后端地址（运行时）
  const apiUrl = process.env.API_URL || "http://localhost:8080";

  // 构建目标 URL
  const targetUrl = new URL(pathname + request.nextUrl.search, apiUrl);

  // 复制请求头
  const headers = new Headers(request.headers);
  // 移除 host 头，让目标服务器使用自己的 host
  headers.delete("host");

  // 代理请求
  return NextResponse.rewrite(targetUrl, {
    request: {
      headers,
    },
  });
}

// 配置 middleware 匹配的路径
export const config = {
  matcher: ["/api/v2/:path*", "/uiapi/:path*", "/ingest/:path*", "/ai/:path*"],
};
