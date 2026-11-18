"use client";

import { useEffect } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useAuthStore } from "@/store/authStore";
import { useClusterStore } from "@/store/clusterStore";
import { clusterListQuery } from "@/queries/session";
import { resetQueriesWhenIdentityChanges } from "@/queries/session-reset";

/**
 * 加载会话级数据：当前主体的权限、已接入的集群。
 *
 * 挂在根部，全站只有一份 —— 页面切换不会重新请求。
 * 权限是前端唯一的权限来源；访客同样会拿到 0 级权限。
 */
export function SessionLoader() {
  const refreshMe = useAuthStore((state) => state.refreshMe);
  useEffect(() => {
    void refreshMe();
  }, [refreshMe]);

  // 换了登录身份，按旧身份取回来的缓存一律作废
  const queryClient = useQueryClient();
  useEffect(() => resetQueriesWhenIdentityChanges(queryClient), [queryClient]);

  const setClusterIds = useClusterStore((state) => state.setClusterIds);
  const setOTelAvailableMap = useClusterStore((state) => state.setOTelAvailableMap);
  const markListLoaded = useClusterStore((state) => state.markListLoaded);
  const { data: clusters, isFetched } = useQuery(clusterListQuery());

  useEffect(() => {
    if (!clusters) return;
    setClusterIds(clusters.map((cluster) => cluster.clusterId));
    setOTelAvailableMap(
      Object.fromEntries(clusters.map((cluster) => [cluster.clusterId, cluster.otelAvailable ?? false])),
    );
  }, [clusters, setClusterIds, setOTelAvailableMap]);

  useEffect(() => {
    if (isFetched) markListLoaded();
  }, [isFetched, markListLoaded]);

  return null;
}
