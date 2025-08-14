import request from "@/utils/request";

/**
 * 获取集群概览成功
 */
export function getClusterOverview() {
  return request({
    url: "/uiapi/cluster/overview",
    method: "get",
  });
}
