import request from '@/utils/request'

export function getAllIngresses() {
  return request({
    url: '/uiapi/ingress/list/all',
    method: 'get'
  })
}
