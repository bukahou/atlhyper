// src/api/config/slack.js
import request from '@/utils/request'

// 读取配置（后端已支持 GET/POST，这里用 GET 更直观）
export function getSlackConfig() {
  return request({
    url: '/uiapi/config/slack/get',
    method: 'post'
  })
}

// 部分更新：只传需要更新的字段（未传的不改）
export function updateSlackConfig(payload = {}) {
  const data = {}
  if (typeof payload.enable !== 'undefined') data.enable = payload.enable // 0/1
  if (typeof payload.webhook !== 'undefined') data.webhook = payload.webhook
  if (typeof payload.intervalSec !== 'undefined') { data.intervalSec = Number(payload.intervalSec) }

  if (Object.keys(data).length === 0) {
    // 与后端“未提供任何可更新字段”逻辑一致，前端也做一次保护
    return Promise.reject(new Error('未提供任何可更新字段'))
  }

  return request({
    url: '/uiapi/config/slack/update',
    method: 'post',
    data
  })
}
