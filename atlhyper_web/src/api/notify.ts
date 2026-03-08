/**
 * 通知配置 API
 *
 * 适配 Master V2 API
 */

import { get, put } from "./request";

// ============================================================
// 类型定义
// ============================================================

// 渠道类型
export type ChannelType = "slack" | "email";

// Slack 配置
export interface SlackConfig {
  webhookUrl: string;
}

// Email 配置
export interface EmailConfig {
  smtpHost: string;
  smtpPort: number;
  smtpUser: string;
  smtpPassword?: string; // 查询时不返回，更新时传入
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
export function listChannels() {
  return get<ListChannelsResponse>("/api/v2/notify/channels");
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
  if (data.webhookUrl !== undefined) {
    req.config = { webhookUrl: data.webhookUrl };
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
 * 测试通知渠道
 * 调用 tester 模块 (端口 9080)
 */
export async function testChannel(type: ChannelType): Promise<{ success: boolean; message: string }> {
  const testerUrl = process.env.NEXT_PUBLIC_TESTER_URL || "http://localhost:9080";

  try {
    const response = await fetch(`${testerUrl}/test/notifier/${type}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
    });

    const data = await response.json();
    return {
      success: response.ok && data.success,
      message: data.message || (response.ok ? "测试消息已发送" : "测试失败"),
    };
  } catch (error) {
    return {
      success: false,
      message: error instanceof Error ? error.message : "网络错误",
    };
  }
}

// ============================================================
// Mock 数据（Guest 用户使用）
// ============================================================

export const mockChannels: NotifyChannel[] = [
  {
    id: 1,
    type: "slack",
    name: "Slack",
    enabled: true,
    effectiveEnabled: true,
    validationErrors: [],
    config: {
      webhookUrl: "https://hooks.slack.com/services/T****/B****/************",
    } as SlackConfig,
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-01-20T10:30:00Z",
  },
  {
    id: 2,
    type: "email",
    name: "Email",
    enabled: false,
    effectiveEnabled: false,
    validationErrors: ["smtpUser 未配置", "smtpPassword 未配置"],
    config: {
      smtpHost: "smtp.example.com",
      smtpPort: 587,
      smtpUser: "",
      fromAddress: "alerts@example.com",
      toAddresses: ["admin@example.com", "ops@example.com"],
    } as EmailConfig,
    createdAt: "2025-01-01T00:00:00Z",
    updatedAt: "2025-01-15T14:20:00Z",
  },
];
