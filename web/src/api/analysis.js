import request from '@/utils/request'

/** 获取集群概览 */
export function getClusterOverview(clusterId) {
  return request({
    url: '/uiapi/cluster/overview',
    method: 'post',
    data: { ClusterID: clusterId }
  })
}
