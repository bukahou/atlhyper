/**
 * 国际化类型定义
 */

import type { Language } from "./common";

// 导航菜单翻译
export interface NavTranslations {
  overview: string;
  workbench: string;
  ai: string;
  cluster: string;
  pod: string;
  node: string;
  deployment: string;
  service: string;
  namespace: string;
  ingress: string;
  alert: string;
  system: string;
  metrics: string;
  logs: string;
  alerts: string;
  account: string;
  users: string;
  roles: string;
  audit: string;
  clusters: string;
  agents: string;
  notifications: string;
}

// 通用翻译
export interface CommonTranslations {
  loading: string;
  error: string;
  success: string;
  confirm: string;
  cancel: string;
  save: string;
  delete: string;
  edit: string;
  search: string;
  refresh: string;
  login: string;
  logout: string;
  username: string;
  password: string;
  submit: string;
  noData: string;
  total: string;
  status: string;
  action: string;
  name: string;
  namespace: string;
  createdAt: string;
}

// 状态翻译
export interface StatusTranslations {
  running: string;
  pending: string;
  failed: string;
  succeeded: string;
  unknown: string;
  ready: string;
  notReady: string;
}

// 审计操作翻译
export interface AuditTranslations {
  // 页面文案
  description: string;
  filterLabel: string;
  timeRange: string;
  user: string;
  result: string;
  all: string;
  successOnly: string;
  failedOnly: string;
  total: string;
  successCount: string;
  failedCount: string;
  noRecords: string;
  // 时间范围选项
  lastHour: string;
  last24Hours: string;
  last7Days: string;
  allTime: string;
  // 操作类型
  actions: {
    login: string;
    logout: string;
    podRestart: string;
    podLogs: string;
    nodeCordon: string;
    nodeUncordon: string;
    deploymentScale: string;
    deploymentUpdateImage: string;
    userRegister: string;
    userUpdateRole: string;
    userDelete: string;
    slackConfigUpdate: string;
    unknown: string;
  };
}

// 完整翻译结构
export interface Translations {
  nav: NavTranslations;
  common: CommonTranslations;
  status: StatusTranslations;
  audit: AuditTranslations;
}

// 国际化上下文
export interface I18nContextType {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: Translations;
}
