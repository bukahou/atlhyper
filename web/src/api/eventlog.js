import request from '@/utils/request'

/**
 * 查询最近异常日志（后端默认返回最近 1 天）
 * @param {number} days 可选参数：指定查询天数
 * @returns {Promise}
 */
export function getRecentEventLogs(days) {
  return request({
    url: '/uiapi/event/list/recent',
    method: 'get',
    params: days ? { days } : {} // 不传参数使用默认 1 天
  })
}
