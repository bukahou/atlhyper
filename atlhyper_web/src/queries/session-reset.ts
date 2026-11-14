/**
 * 登录身份变化时丢弃已缓存的数据
 */
import type { QueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/store/authStore";

/**
 * 登录、登出或登录态失效（token 变化）时，把所有查询重置：
 * 缓存里的数据是按上一个身份的权限取回来的，换了身份就不能再显示；
 * 正在显示的查询会按新身份重新取数。
 *
 * 返回取消订阅的函数。
 */
export function resetQueriesWhenIdentityChanges(queryClient: QueryClient): () => void {
  return useAuthStore.subscribe((state, previous) => {
    if (state.token !== previous.token) {
      void queryClient.resetQueries();
    }
  });
}
