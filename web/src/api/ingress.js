import request from "@/utils/request";

export function getAllIngresses(clusterId) {
  return request({
    url: "/uiapi/ingress/overview",
    method: "post",
    data: { ClusterID: clusterId },
  });
}

export function getIngressesDetail(clusterId, namespace, name) {
  return request({
    url: "/uiapi/ingress/detail",
    method: "post",
    data: {
      ClusterID: clusterId,
      Namespace: namespace,
      Name: name,
    },
  });
}
