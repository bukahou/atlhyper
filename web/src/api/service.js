// src/api/service.js
import request from '@/utils/request'

/**
 * 获取所有 Service 列表
 * GET /uiapi/service/list/all
 */
export function getAllServices() {
  return request({
    url: '/uiapi/service/list/all',
    method: 'get'
  })
}
