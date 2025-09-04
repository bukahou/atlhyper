import request from '@/utils/request'

export function getMetricsOverview(clusterId) {
  return request({
    url: '/uiapi/metrics/overview',
    method: 'post',
    data: { ClusterID: clusterId }
  })
}

export function getMetricsdetail(clusterId, nodeID) {
  return request({
    url: '/uiapi/metrics/node/detail',
    method: 'post',
    data: {
      ClusterID: clusterId,
      NodeID: nodeID
    }
  })
}
