import request from "@/utils/request";

/**
 * 获取 Pod 状态概要
 */
export function getPodSummary() {
  return request({
    url: "/uiapi/pod/summary",
    method: "get",
  });
}

/**
 * 获取 Pod 简要列表
 */
export function getBriefPods() {
  return request({
    url: "/uiapi/pod/list/brief",
    method: "get",
  });
}

/**
 * 获取 Pod 详情信息
 * @param {string} namespace 命名空间
 * @param {string} name Pod 名称
 * @returns Promise
 */
export function getPodDescribe(namespace, name) {
  return request({
    url: `/uiapi/pod/describe/${namespace}/${name}`,
    method: "get",
  });
}

/**
 * 重启 Pod
 * @param {string} namespace
 * @param {string} name
 */
export function restartPod(namespace, name) {
  return request({
    url: `/uiapi/pod-ops/restart/${namespace}/${name}`,
    method: "post",
  });
}
