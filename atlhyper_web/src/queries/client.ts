/**
 * 全站共用的 QueryClient 与默认行为
 *
 * 读请求一律经 TanStack Query：同一份数据只请求一次、后发的结果生效、
 * 参数变化或组件卸载时取消请求。各查询只在 queries/ 下的工厂里声明自己的差异。
 */
import { QueryClient } from "@tanstack/react-query";
import { isAxiosError } from "axios";

/** 读请求失败后是否重试：网络错误与 5xx 重试一次；4xx 重发也不会变，不重试 */
export function shouldRetry(failureCount: number, error: unknown): boolean {
  if (isAxiosError(error)) {
    const status = error.response?.status;
    if (status !== undefined && status >= 400 && status < 500) return false;
  }
  return failureCount < 1;
}

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: shouldRetry,
        // 标签页不可见时停止轮询，切回时立即刷新一次
        refetchIntervalInBackground: false,
        refetchOnWindowFocus: true,
      },
    },
  });
}
