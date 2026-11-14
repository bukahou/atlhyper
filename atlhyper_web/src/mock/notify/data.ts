/**
 * 通知设置页的演示数据（无 notify.manage 权限时整页展示，均为虚构）
 */
import type { EmailConfig, NotifyChannel, SlackConfig } from "@/api/notify";

export const mockChannels: NotifyChannel[] = [
  {
    id: 1,
    type: "slack",
    name: "Slack",
    enabled: true,
    effectiveEnabled: true,
    validationErrors: [],
    config: { webhookUrl: "" } as SlackConfig,
    configuredSecrets: ["webhookUrl"],
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
