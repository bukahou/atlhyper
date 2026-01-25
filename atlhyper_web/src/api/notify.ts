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
  webhook_url: string;
}

// Email 配置
export interface EmailConfig {
  smtp_host: string;
  smtp_port: number;
  smtp_user: string;
  smtp_password?: string; // 查询时不返回，更新时传入
  smtp_tls?: boolean;
  from_address: string;
  to_addresses: string[];
}

// 通知渠道
export interface NotifyChannel {
  id: number;
  type: ChannelType;
  name: string;
  enabled: boolean;
  effective_enabled: boolean; // 实际可用状态（启用+配置完整）
  validation_errors: string[]; // 配置校验错误
  config: SlackConfig | EmailConfig;
  created_at: string;
  updated_at: string;
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
export function updateSlack(data: { enabled?: boolean; webhook_url?: string }) {
  const req: UpdateChannelRequest = {};
  if (data.enabled !== undefined) {
    req.enabled = data.enabled;
  }
  if (data.webhook_url !== undefined) {
    req.config = { webhook_url: data.webhook_url };
  }
  return put<NotifyChannel>("/api/v2/notify/channels/slack", req);
}

// Email 更新参数
export interface EmailUpdateData {
  enabled?: boolean;
  smtp_host?: string;
  smtp_port?: number;
  smtp_user?: string;
  smtp_password?: string;
  smtp_tls?: boolean;
  from_address?: string;
  to_addresses?: string[];
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
  if (data.smtp_host !== undefined) config.smtp_host = data.smtp_host;
  if (data.smtp_port !== undefined) config.smtp_port = data.smtp_port;
  if (data.smtp_user !== undefined) config.smtp_user = data.smtp_user;
  if (data.smtp_password !== undefined) config.smtp_password = data.smtp_password;
  if (data.smtp_tls !== undefined) config.smtp_tls = data.smtp_tls;
  if (data.from_address !== undefined) config.from_address = data.from_address;
  if (data.to_addresses !== undefined) config.to_addresses = data.to_addresses;

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
    effective_enabled: true,
    validation_errors: [],
    config: {
      webhook_url: "https://hooks.slack.com/services/T****/B****/************",
    } as SlackConfig,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-20T10:30:00Z",
  },
  {
    id: 2,
    type: "email",
    name: "Email",
    enabled: false,
    effective_enabled: false,
    validation_errors: ["smtp_user 未配置", "smtp_password 未配置"],
    config: {
      smtp_host: "smtp.example.com",
      smtp_port: 587,
      smtp_user: "",
      from_address: "alerts@example.com",
      to_addresses: ["admin@example.com", "ops@example.com"],
    } as EmailConfig,
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-15T14:20:00Z",
  },
];
