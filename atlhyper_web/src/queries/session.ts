/**
 * 会话级数据的取数描述（全站只需要一份）
 */
import { mutationOptions, queryOptions } from "@tanstack/react-query";
import { getAuthProviders, login } from "@/api/auth";
import { getClusterList } from "@/api/cluster";
import type { LoginRequest } from "@/types/auth";

// 外部登录的发起地址：只是拼一个地址（整页跳转用），不发请求。登录框从这里取，不直接碰 api 层
export { oidcStartURL } from "@/api/auth";
import { sessionKeys } from "./keys";
import { REFRESH } from "./refresh";

/** 已接入的集群列表。慢速轮询：新接入的集群与 OTel 可用性变化会自己出现 */
export function clusterListQuery() {
  return queryOptions({
    queryKey: sessionKeys.clusterList(),
    queryFn: async ({ signal }) => (await getClusterList(signal)).data?.clusters ?? [],
    refetchInterval: REFRESH.slow,
    staleTime: REFRESH.slow,
  });
}

/**
 * 外部登录方式（未启用时为空列表）。只在登录框打开时取；
 * 一次会话里不会变，取过就不再重取。
 */
export function authProvidersQuery(loginDialogOpen: boolean) {
  return queryOptions({
    queryKey: sessionKeys.authProviders(),
    queryFn: async ({ signal }) => (await getAuthProviders(signal)).data.data ?? [],
    staleTime: Infinity,
    enabled: loginDialogOpen,
  });
}

/** 用户名密码登录。返回 token、用户与可见集群，由调用方写进 authStore */
export function loginMutation() {
  return mutationOptions({
    mutationFn: async (credentials: LoginRequest) => (await login(credentials)).data.data,
  });
}
