/**
 * 通知模块的数据类型
 *
 * 字段名与取值以 docs/design/active/incident-notification-design.md 的接口契约为准，
 * 前端直接使用后端返回的字段，不做重命名。时间均为 RFC 3339 字符串；「没有」用空串表示。
 */

/** 严重度四档，与 AIOps 事件一致；顺序从低到高 */
export type NotifySeverity = "low" | "medium" | "high" | "critical";
export const NOTIFY_SEVERITIES: readonly NotifySeverity[] = ["low", "medium", "high", "critical"];

/** 告警来源：AIOps 事件 / Agent 心跳 / 旧的逐条 K8s Warning 事件 */
export type NotifyAlertSource = "incident" | "heartbeat" | "k8s_event";
export const NOTIFY_ALERT_SOURCES: readonly NotifyAlertSource[] = ["incident", "heartbeat", "k8s_event"];

// ============================================================
// 通知策略
// ============================================================

/** 全局通知策略（单行）。GET / PUT /api/v2/notify/policy 的 data */
export interface NotifyPolicy {
  incidentAlertsEnabled: boolean;
  /** 低于此严重度的告警不通知 */
  minSeverity: NotifySeverity;
  notifyResolved: boolean;
  /** 警告级事件 24 小时内发生几次才通知；0 = 不通知 */
  warningFlapThreshold: number;
  groupWaitSeconds: number;
  groupIntervalSeconds: number;
  repeatIntervalSeconds: number;
  /** 旧的逐条 K8s 事件告警 */
  k8sEventAlertsEnabled: boolean;
  /** 没有规则命中时发给谁；空 = 全部已启用的接收人 */
  defaultReceiverIds: number[];
}

// ============================================================
// 告警与送达记录
// ============================================================

export type NotifyAlertStatus = "firing" | "resolved";
/** 告警最近一次变化 */
export type NotifyAlertKind = "firing" | "escalated" | "recurred" | "flapping" | "resolved";

export interface NotifyAlert {
  fingerprint: string;
  source: NotifyAlertSource;
  clusterId: string;
  severity: NotifySeverity;
  status: NotifyAlertStatus;
  kind: NotifyAlertKind;
  title: string;
  summary: string;
  /** 主体键，如 shop/deployment/api；心跳告警没有主体，为空串 */
  subject: string;
  /** 对应的 AIOps 事件号；不是事件来源的告警为空串 */
  incidentId: string;
  startsAt: string;
  /** 仍在持续时为空串 */
  endsAt: string;
  /** 还没发过通知时为空串 */
  lastNotifiedAt: string;
  notifyCount: number;
  /** 被什么抑制，如 "cluster-offline"；没有被抑制为空串 */
  suppressedBy: string;
  /** 命中的静默的 id；没有被静默为 0 */
  silencedBy: number;
}

/** 一条送达记录对应的变化；repeat = 重复提醒 */
export type NotifyDeliveryKind = NotifyAlertKind | "repeat";
export type NotifyDeliveryStatus = "pending" | "success" | "failed" | "abandoned" | "no-receiver";
export const NOTIFY_DELIVERY_STATUSES: readonly NotifyDeliveryStatus[] = [
  "pending",
  "success",
  "failed",
  "abandoned",
  "no-receiver",
];

export interface NotifyDelivery {
  id: number;
  batchId: string;
  fingerprint: string;
  alertTitle: string;
  /** 接收人删除后记录仍可读，所以名称与类型随记录保存 */
  receiverId: number;
  receiverName: string;
  receiverType: string;
  kind: NotifyDeliveryKind;
  status: NotifyDeliveryStatus;
  attempts: number;
  lastError: string;
  createdAt: string;
  updatedAt: string;
}

/** 一页结果：列表 + 符合条件的总数 */
export interface NotifyPage<T> {
  items: T[];
  total: number;
}

// ============================================================
// 接收人
// ============================================================

export type NotifyReceiverType = "slack" | "email";

/**
 * 邮件接收人的设置：只有收件人。
 * 从哪台服务器、用哪个账号发由全局发件设置决定（见下方 NotifySMTP）。
 */
export interface NotifyEmailRecipients {
  toAddresses: string[];
}

/**
 * 接收人（列表项）。凭据只写不读：Webhook 地址永不返回，只返回「是否已配置」。
 */
export interface NotifyReceiver {
  id: number;
  name: string;
  type: NotifyReceiverType;
  enabled: boolean;
  slack?: { webhookConfigured: boolean };
  email?: NotifyEmailRecipients;
  createdAt: string;
  updatedAt: string;
}

/**
 * 新建 / 修改接收人的请求体。
 *
 * ⛔ webhookUrl 是凭据明文：只在发请求的那一刻存在，不得放进查询缓存。
 * 修改时留空 = 保持原值。邮件接收人没有凭据。
 */
export type NotifyReceiverBody =
  | { name: string; type: "slack"; enabled: boolean; slack: { webhookUrl: string } }
  | { name: string; type: "email"; enabled: boolean; email: NotifyEmailRecipients };

// ============================================================
// 全局发件设置（邮件接收人共用）
// ============================================================

/** 发件设置里可以读回、可以留存的部分（不含密码） */
export interface NotifySMTPSettings {
  host: string;
  port: number;
  user: string;
  tls: boolean;
  /** 空 = 使用 user 作为发件地址 */
  fromAddress: string;
}

/** GET / PUT /api/v2/notify/smtp 的 data。密码只写不读，只返回「是否已配置」 */
export interface NotifySMTP extends NotifySMTPSettings {
  /** 主机、端口、账号、密码都已配置 */
  configured: boolean;
  passwordConfigured: boolean;
  /** 从没保存过 = null（与本文件「没有用空串」的约定不同，以接口契约为准） */
  updatedAt: string | null;
}

/**
 * PUT /api/v2/notify/smtp 的请求体。
 *
 * ⛔ password 是凭据明文：只在发请求的那一刻存在，不得放进查询缓存。
 * 留空 = 保持原值；第一次保存必须填写（否则后端 400）。
 */
export interface NotifySMTPBody extends NotifySMTPSettings {
  password: string;
}

// ============================================================
// 路由规则
// ============================================================

/** 规则里可以编辑的部分（POST / PUT 的请求体） */
export interface NotifyRuleBody {
  name: string;
  enabled: boolean;
  /** 空 = 全部集群 */
  clusterIds: string[];
  /** 空 = 全部来源 */
  sources: NotifyAlertSource[];
  minSeverity: NotifySeverity;
  receiverIds: number[];
  /** 三个时间参数为 null 时用策略里的全局值 */
  groupWaitSeconds: number | null;
  groupIntervalSeconds: number | null;
  repeatIntervalSeconds: number | null;
}

export interface NotifyRule extends NotifyRuleBody {
  id: number;
  /** 匹配顺序：从小到大，第一条命中的生效 */
  position: number;
  createdAt: string;
  updatedAt: string;
}

/** 试匹配的输入：假想一条告警 */
export interface NotifyRuleMatchRequest {
  clusterId: string;
  source: NotifyAlertSource;
  severity: NotifySeverity;
}

export interface NotifyRuleMatchResult {
  /** null = 没有规则命中，走默认策略 */
  matchedRule: { id: number; name: string } | null;
  receivers: { id: number; name: string; type: NotifyReceiverType; enabled: boolean }[];
  /** 默认策略下严重度不到 minSeverity，不会发送 */
  belowMinSeverity: boolean;
}

// ============================================================
// 静默
// ============================================================

/** pending = 未开始，active = 生效中，expired = 已过期 */
export type NotifySilenceState = "pending" | "active" | "expired";

/** 新建静默的请求体。三个范围条件留空 = 不限 */
export interface NotifySilenceBody {
  clusterId: string;
  namespace: string;
  /** 填了则精确匹配主体键 */
  subject: string;
  startsAt: string;
  endsAt: string;
  comment: string;
}

export interface NotifySilence extends NotifySilenceBody {
  id: number;
  createdBy: string;
  createdAt: string;
  state: NotifySilenceState;
}
