import request from "@/utils/request";

export function getNodeOverview(clusterId) {
  return request({
    url: "/uiapi/node/overview",
    method: "post",
    data: { ClusterID: clusterId },
  });
}

export function getNodeDetail(clusterId, nodename) {
  return request({
    url: "/uiapi/node/detail",
    method: "post",
    data: {
      ClusterID: clusterId,
      NodeName: nodename,
    },
  });
}

export function getNodecordon(clusterId, nodename) {
  return request({
    url: "/uiapi/ops/node/cordon",
    method: "post",
    data: {
      ClusterID: clusterId,
      Node: nodename,
    },
  });
}

/** 节点解封（uncordon）——后端参数名为大写 Node */
export function getNodeuncordon(clusterId, nodename) {
  return request({
    url: "/uiapi/ops/node/uncordon",
    method: "post",
    data: {
      ClusterID: clusterId,
      Node: nodename,
    },
  });
}
