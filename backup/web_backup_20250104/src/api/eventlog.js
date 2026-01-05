import request from '@/utils/request'

export function getRecentEventLogs(clusterId, withinDays) {
  return request({
    url: '/uiapi/event/logs',
    method: 'post',
    data: {
      ClusterID: clusterId,
      WithinDays: withinDays
    }
  })
}
