import request from "@/utils/request";

export function getAllNamespaces(clusterId) {
  return request({
    url: "/uiapi/namespace/overview",
    method: "post",
    data: { ClusterID: clusterId },
  });
}

export function getNamespacesDetail(clusterId, namespace) {
  return request({
    url: "/uiapi/namespace/detail",
    method: "post",
    data: {
      ClusterID: clusterId,
      Namespace: namespace,
    },
  });
}
