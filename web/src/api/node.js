import request from "@/utils/request";

/**
 * 获取节点总览（统计信息 + 简要节点列表）
 * GET /uiapi/node/overview
 */
export function getNodeOverview() {
  return request({
    url: "/uiapi/node/overview",
    method: "get",
  });
}

/**
 * 设置节点调度状态（封锁 / 解封）
 * POST /uiapi/node/schedulable
 * @param {string} name 节点名称
 * @param {boolean} unschedulable true 表示封锁，false 表示解封
 */
export function setNodeSchedulable(name, unschedulable) {
  return request({
    url: "/uiapi/node-ops/schedule",
    method: "post",
    data: {
      name,
      unschedulable,
    },
  });
}

/**
 * 获取指定 Node 的详细信息
 * GET /uiapi/node/get/:name
 * @param {string} name 节点名称
 */
export function getNodeDetail(name) {
  return request({
    url: `/uiapi/node/get/${name}`,
    method: "get",
  });
}
