/**
 * 通知设置页的翻译类型（六个页签）
 *
 * 从 i18n.ts 拆出来单放：这一页的文案量大，i18n.ts 已经很长。
 * 带 {xxx} 的文案是模板，由组件用 replace 填入。
 */
import type { NotifyWriteFailure } from "@/lib/notify-failure";
import type { NotifyRuleProblem } from "@/lib/notify-rule-input";
import type { NotifySilenceProblem } from "@/lib/notify-silence-input";
import type {
  NotifyAlertSource,
  NotifyDeliveryKind,
  NotifyDeliveryStatus,
  NotifySeverity,
  NotifySilenceState,
} from "./notify";

/** 接收人表单里可能出现的问题 */
export type NotifyReceiverProblemKey =
  | "nameRequired"
  | "webhookRequired"
  | "smtpHostRequired"
  | "smtpPortInvalid"
  | "smtpUserRequired"
  | "passwordRequired"
  | "recipientsRequired";

export interface NotifyAlertsTabTranslations {
  statusFiring: string;
  statusResolved: string;
  severity: string;
  kind: string;
  title: string;
  subject: string;
  cluster: string;
  startsAt: string;
  endsAt: string;
  notifyCount: string;
  lastNotifiedAt: string;
  neverNotified: string;
  suppressed: string;
  suppressedByClusterOffline: string;
  /** {reason} */
  suppressedByOther: string;
  silenced: string;
  /** {id} */
  silencedHint: string;
  viewIncident: string;
  emptyFiring: string;
  emptyResolved: string;
}

export interface NotifyDeliveriesTabTranslations {
  alertTitle: string;
  receiver: string;
  kind: string;
  status: string;
  attempts: string;
  lastError: string;
  createdAt: string;
  allStatuses: string;
  statusLabel: Record<NotifyDeliveryStatus, string>;
  empty: string;
}

export interface NotifyReceiversTabTranslations {
  add: string;
  createTitle: string;
  editTitle: string;
  name: string;
  namePlaceholder: string;
  type: string;
  typeSlack: string;
  typeEmail: string;
  target: string;
  enabled: string;
  updatedAt: string;
  webhookUrl: string;
  webhookUrlHint: string;
  webhookConfigured: string;
  webhookNotConfigured: string;
  passwordNotConfigured: string;
  /** 编辑时凭据输入框的提示：已配置，留空则保持不变 */
  credentialKeepHint: string;
  /** {count} */
  recipientsCount: string;
  test: string;
  testSent: string;
  createdToast: string;
  updatedToast: string;
  deletedToast: string;
  deleteConfirmTitle: string;
  /** {name} */
  deleteConfirmMessage: string;
  empty: string;
  emptyHint: string;
  problems: Record<NotifyReceiverProblemKey, string>;
  // 邮件表单（EmailFormFields / TagInput 的文案）
  smtpServer: string;
  smtpServerPlaceholder: string;
  smtpCustom: string;
  smtpCustomPlaceholder: string;
  port: string;
  portHint: string;
  emailAccount: string;
  emailAccountHint: string;
  password: string;
  passwordPlaceholder: string;
  tlsEncryption: string;
  tlsHint: string;
  recipients: string;
  recipientsPlaceholder: string;
  tagInputDuplicate: string;
  tagInputInvalidFormat: string;
  smtpPresets: {
    qq: string;
    netease163: string;
    netease126: string;
    tencentEnterprise: string;
    aliEnterprise: string;
    feishu: string;
  };
}

export interface NotifyRulesTabTranslations {
  add: string;
  createTitle: string;
  editTitle: string;
  orderHint: string;
  position: string;
  name: string;
  enabled: string;
  conditions: string;
  clusters: string;
  clustersHint: string;
  allClusters: string;
  addClusterPlaceholder: string;
  sources: string;
  sourcesHint: string;
  allSources: string;
  minSeverity: string;
  /** {severity} */
  minSeverityAtLeast: string;
  receivers: string;
  noReceivers: string;
  noReceiversHint: string;
  /** {id} —— 规则引用的接收人不在列表里时的兜底显示 */
  unknownReceiver: string;
  timing: string;
  timingHint: string;
  timingUseGlobal: string;
  groupWaitSeconds: string;
  groupIntervalSeconds: string;
  repeatIntervalSeconds: string;
  moveUp: string;
  moveDown: string;
  createdToast: string;
  updatedToast: string;
  deletedToast: string;
  deleteConfirmTitle: string;
  /** {name} */
  deleteConfirmMessage: string;
  empty: string;
  emptyHint: string;
  /** 时间参数的问题用 rangeError（见 NotificationsTranslations）；这里只有名称 */
  problems: Record<Extract<NotifyRuleProblem, "nameRequired">, string>;
  // 试匹配
  matchTitle: string;
  matchHint: string;
  matchCluster: string;
  matchClusterPlaceholder: string;
  matchSource: string;
  matchSeverity: string;
  matchRun: string;
  matchedRule: string;
  matchedDefault: string;
  matchReceivers: string;
  matchNoReceivers: string;
  matchBelowMinSeverity: string;
  matchReceiverDisabled: string;
}

export interface NotifySilencesTabTranslations {
  stateLabel: Record<NotifySilenceState, string>;
  add: string;
  createTitle: string;
  scope: string;
  cluster: string;
  allClusters: string;
  namespace: string;
  allNamespaces: string;
  namespacePlaceholder: string;
  subject: string;
  allSubjects: string;
  subjectPlaceholder: string;
  subjectHint: string;
  startsAt: string;
  endsAt: string;
  comment: string;
  commentPlaceholder: string;
  createdBy: string;
  heartbeatHint: string;
  allAlertsWarning: string;
  expire: string;
  expireConfirmTitle: string;
  expireConfirmMessage: string;
  createdToast: string;
  expiredToast: string;
  empty: Record<NotifySilenceState, string>;
  problems: Record<NotifySilenceProblem, string>;
}

export interface NotifyPolicyTabTranslations {
  incidentSection: string;
  timingSection: string;
  legacySection: string;
  incidentAlertsEnabled: string;
  incidentAlertsEnabledHint: string;
  minSeverity: string;
  minSeverityHint: string;
  notifyResolved: string;
  notifyResolvedHint: string;
  warningFlapThreshold: string;
  warningFlapThresholdHint: string;
  groupWaitSeconds: string;
  groupWaitSecondsHint: string;
  groupIntervalSeconds: string;
  groupIntervalSecondsHint: string;
  repeatIntervalSeconds: string;
  repeatIntervalSecondsHint: string;
  k8sEventAlertsEnabled: string;
  k8sEventAlertsEnabledHint: string;
  defaultReceivers: string;
  defaultReceiversHint: string;
  noReceiversYet: string;
  savedToast: string;
}

export interface NotificationsTranslations {
  pageDescription: string;
  demoHint: string;
  tabs: {
    alerts: string;
    deliveries: string;
    receivers: string;
    rules: string;
    silences: string;
    policy: string;
  };
  severity: Record<NotifySeverity, string>;
  source: Record<NotifyAlertSource, string>;
  /** 告警 / 送达记录对应的变化 */
  kind: Record<NotifyDeliveryKind, string>;
  /** 写操作失败的提示，按失败原因 */
  errors: Record<NotifyWriteFailure, string>;
  enabledLabel: string;
  disabledLabel: string;
  seconds: string;
  /** {min} {max} */
  rangeError: string;
  saving: string;
  alerts: NotifyAlertsTabTranslations;
  deliveries: NotifyDeliveriesTabTranslations;
  receivers: NotifyReceiversTabTranslations;
  rules: NotifyRulesTabTranslations;
  silences: NotifySilencesTabTranslations;
  policy: NotifyPolicyTabTranslations;
}
