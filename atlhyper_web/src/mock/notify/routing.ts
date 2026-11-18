/**
 * 通知设置页的演示数据：策略、接收人、路由规则、静默（没有 notify.manage 权限时整页展示，均为虚构）
 *
 * 字段与 types/notify.ts 一致。
 * 接收人与发件设置只有「凭据是否已配置」，没有任何凭据值。
 */
import type { NotifyPolicy, NotifyReceiver, NotifyRule, NotifySilence, NotifySMTP } from "@/types/notify";

/** 相对现在偏移若干分钟的时刻（负数 = 过去）：静默的状态由起止时间决定，必须相对现在 */
function minutesFromNow(minutes: number): string {
  return new Date(Date.now() + minutes * 60_000).toISOString();
}

/** 演示用的集群：规则与静默的集群选项在演示模式下用它，不读真实的集群列表 */
export const MOCK_NOTIFY_CLUSTER_IDS: string[] = ["prod", "staging"];

/** 策略：后端的默认值 */
export const MOCK_NOTIFY_POLICY: NotifyPolicy = {
  incidentAlertsEnabled: true,
  minSeverity: "low",
  notifyResolved: true,
  warningFlapThreshold: 3,
  groupWaitSeconds: 30,
  groupIntervalSeconds: 300,
  repeatIntervalSeconds: 14400,
  k8sEventAlertsEnabled: false,
  defaultReceiverIds: [],
};

export const MOCK_NOTIFY_RECEIVERS: NotifyReceiver[] = [
  {
    id: 1,
    name: "On-call channel",
    type: "slack",
    enabled: true,
    slack: { webhookConfigured: true },
    createdAt: "2026-10-01T09:00:00Z",
    updatedAt: "2026-10-01T09:00:00Z",
  },
  {
    id: 2,
    name: "Ops mail",
    type: "email",
    enabled: true,
    email: { toAddresses: ["oncall@example.com", "sre@example.com"] },
    createdAt: "2026-10-01T09:05:00Z",
    updatedAt: "2026-10-20T02:30:00Z",
  },
  {
    id: 3,
    name: "Dev channel",
    type: "slack",
    enabled: false,
    slack: { webhookConfigured: false },
    createdAt: "2026-10-12T04:00:00Z",
    updatedAt: "2026-10-12T04:00:00Z",
  },
];

/** 全局发件设置：已配置，发件地址留空（= 使用账号） */
export const MOCK_NOTIFY_SMTP: NotifySMTP = {
  configured: true,
  host: "smtp.example.com",
  port: 587,
  user: "ops@example.com",
  tls: true,
  fromAddress: "",
  passwordConfigured: true,
  updatedAt: "2026-10-01T08:55:00Z",
};

export const MOCK_NOTIFY_RULES: NotifyRule[] = [
  {
    id: 1,
    name: "Production, high and above",
    position: 1,
    enabled: true,
    clusterIds: ["prod"],
    sources: ["incident"],
    minSeverity: "high",
    receiverIds: [1, 2],
    groupWaitSeconds: null,
    groupIntervalSeconds: null,
    repeatIntervalSeconds: 3600,
    createdAt: "2026-10-01T09:10:00Z",
    updatedAt: "2026-10-01T09:10:00Z",
  },
  {
    id: 2,
    name: "Agent offline",
    position: 2,
    enabled: true,
    clusterIds: [],
    sources: ["heartbeat"],
    minSeverity: "low",
    receiverIds: [1],
    groupWaitSeconds: 0,
    groupIntervalSeconds: null,
    repeatIntervalSeconds: null,
    createdAt: "2026-10-01T09:12:00Z",
    updatedAt: "2026-10-01T09:12:00Z",
  },
  {
    id: 3,
    name: "Staging to dev channel",
    position: 3,
    enabled: false,
    clusterIds: ["staging"],
    sources: [],
    minSeverity: "medium",
    receiverIds: [3],
    groupWaitSeconds: null,
    groupIntervalSeconds: null,
    repeatIntervalSeconds: null,
    createdAt: "2026-10-12T04:05:00Z",
    updatedAt: "2026-10-12T04:05:00Z",
  },
];

export const MOCK_NOTIFY_SILENCES: NotifySilence[] = [
  {
    id: 1,
    clusterId: "prod",
    namespace: "shop",
    subject: "shop/statefulset/orders-db",
    startsAt: minutesFromNow(-40),
    endsAt: minutesFromNow(80),
    comment: "Database upgrade",
    createdBy: "admin",
    createdAt: minutesFromNow(-45),
    state: "active",
  },
  {
    id: 2,
    clusterId: "staging",
    namespace: "",
    subject: "",
    startsAt: minutesFromNow(60 * 20),
    endsAt: minutesFromNow(60 * 24),
    comment: "Node pool replacement",
    createdBy: "admin",
    createdAt: minutesFromNow(-120),
    state: "pending",
  },
  {
    id: 3,
    clusterId: "",
    namespace: "kube-system",
    subject: "",
    startsAt: minutesFromNow(-60 * 30),
    endsAt: minutesFromNow(-60 * 28),
    comment: "Control plane maintenance",
    createdBy: "admin",
    createdAt: minutesFromNow(-60 * 31),
    state: "expired",
  },
];
