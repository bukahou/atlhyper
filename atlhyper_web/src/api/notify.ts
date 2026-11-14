/**
 * 通知配置 API
 *
 * 适配 Master V2 API
 */

import { get, post, put } from "./request";

// ============================================================
// 类型定义
// ============================================================

// 渠道类型
export type ChannelType = "slack" | "email";

// Slack 配置
export interface SlackConfig {
  webhookUrl: string; // 凭据：查询时恒为空（是否已设置看 configuredSecrets），更新时传入新值
}

// Email 配置
export interface EmailConfig {
  smtpHost: string;
  smtpPort: number;
  smtpUser: string;
  smtpPassword?: string; // 凭据：查询时恒为空（是否已设置看 configuredSecrets），更新时传入新值
  smtpTLS?: boolean;
  fromAddress: string;
  toAddresses: string[];
}

// 通知渠道
export interface NotifyChannel {
  id: number;
  type: ChannelType;
  name: string;
  enabled: boolean;
  effectiveEnabled: boolean; // 实际可用状态（启用+配置完整）
  validationErrors: string[]; // 配置校验错误
  config: SlackConfig | EmailConfig;
  configuredSecrets?: string[]; // 已设置的凭据字段名（值不回显）；保存时不传 = 保持不变
  createdAt: string;
  updatedAt: string;
}

// 列表响应
interface ListChannelsResponse {
  channels: NotifyChannel[];
  total: number;
}

// 更新请求
interface UpdateChannelRequest {
  enabled?: boolean;
  name?: string;
  config?: Record<string, unknown>;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取所有通知渠道
 * GET /api/v2/notify/channels
 */
export function listChannels(signal?: AbortSignal) {
  return get<ListChannelsResponse>("/api/v2/notify/channels", undefined, signal);
}

/**
 * 获取指定渠道详情
 * GET /api/v2/notify/channels/{type}
 */
export function getChannel(type: ChannelType) {
  return get<NotifyChannel>(`/api/v2/notify/channels/${type}`);
}

/**
 * 更新 Slack 配置
 * PUT /api/v2/notify/channels/slack
 */
export function updateSlack(data: { enabled?: boolean; webhookUrl?: string }) {
  const req: UpdateChannelRequest = {};
  if (data.enabled !== undefined) {
    req.enabled = data.enabled;
  }
  if (data.webhookUrl) {
    req.config = { webhookUrl: data.webhookUrl }; // 留空 = 保持已设置的 webhook 不变
  }
  return put<NotifyChannel>("/api/v2/notify/channels/slack", req);
}

// Email 更新参数
export interface EmailUpdateData {
  enabled?: boolean;
  smtpHost?: string;
  smtpPort?: number;
  smtpUser?: string;
  smtpPassword?: string;
  smtpTLS?: boolean;
  fromAddress?: string;
  toAddresses?: string[];
}

/**
 * 更新 Email 配置
 * PUT /api/v2/notify/channels/email
 */
export function updateEmail(data: EmailUpdateData) {
  const req: UpdateChannelRequest = {};
  if (data.enabled !== undefined) {
    req.enabled = data.enabled;
  }

  // 构建 config 对象（只包含有值的字段）
  const config: Record<string, unknown> = {};
  if (data.smtpHost !== undefined) config.smtpHost = data.smtpHost;
  if (data.smtpPort !== undefined) config.smtpPort = data.smtpPort;
  if (data.smtpUser !== undefined) config.smtpUser = data.smtpUser;
  if (data.smtpPassword !== undefined) config.smtpPassword = data.smtpPassword;
  if (data.smtpTLS !== undefined) config.smtpTLS = data.smtpTLS;
  if (data.fromAddress !== undefined) config.fromAddress = data.fromAddress;
  if (data.toAddresses !== undefined) config.toAddresses = data.toAddresses;

  if (Object.keys(config).length > 0) {
    req.config = config;
  }

  return put<NotifyChannel>("/api/v2/notify/channels/email", req);
}

/**
 * 测试通知渠道：按已保存的配置发一条测试消息
 * POST /api/v2/notify/channels/{type}/test
 */
export async function testChannel(type: ChannelType): Promise<{ success: boolean; message: string }> {
  try {
    const response = await post<{ success: boolean; message: string }>(`/api/v2/notify/channels/${type}/test`);
    return { success: true, message: response.data.message };
  } catch (error) {
    const err = error as { response?: { data?: { error?: string } }; message?: string };
    return { success: false, message: err.response?.data?.error || err.message || "测试失败" };
  }
}
