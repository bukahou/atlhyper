import request from "@/utils/request";

/**
 * 获取所有命名空间及其 Pod 数量等信息
 * GET /uiapi/namespace/list/all
 */
export function getAllNamespaces() {
  return request({
    url: "/uiapi/namespace/list",
    method: "get",
  });
}

/**
 * 获取指定命名空间下的 ConfigMap 列表
 * @param {string} namespace 命名空间名称
 */
export function getConfigMapsByNamespace(namespace) {
  return request({
    url: `/uiapi/configmap/list/by-namespace/${namespace}`,
    method: "get",
  });
}
