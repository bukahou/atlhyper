import request from "@/utils/request";

export function getAllServices(clusterId) {
  return request({
    url: "/uiapi/service/overview",
    method: "post",
    data: { ClusterID: clusterId },
  });
}

export function getServiceDetails(clusterId, namespace, name) {
  return request({
    url: "/uiapi/service/detail",
    method: "post",
    data: {
      ClusterID: clusterId,
      namespace: namespace,
      name: name,
    },
  });
}
