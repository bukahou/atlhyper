import request from '@/utils/request'

export function getLatestMetrics() {
  return request({
    url: '/uiapi/metrics/latest',
    method: 'get'
  })
}
