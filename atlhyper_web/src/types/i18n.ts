/**
 * 国际化类型定义
 */

import type { Language } from "./common";

// 导航菜单翻译
export interface NavTranslations {
  overview: string;
  stylePreview: string;
  slo: string;
  workbench: string;
  workbenchHome: string;
  ai: string;
  commands: string;
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
  settings: string;
  notifications: string;
  aiSettings: string;
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
  // 主题
  themeLight: string;
  themeDark: string;
  themeSystem: string;
  // 侧栏
  collapseSidebar: string;
  expandSidebar: string;
  settings: string;
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
  close: string;
  view: string;
  details: string;
  filter: string;
  clearAll: string;
  all: string;
  enabled: string;
  disabled: string;
  add: string;
  update: string;
  test: string;
  copy: string;
  copied: string;
  download: string;
  upload: string;
  required: string;
  optional: string;
  yes: string;
  no: string;
  on: string;
  off: string;
  back: string;
  next: string;
  previous: string;
  finish: string;
  reset: string;
  apply: string;
  select: string;
  selected: string;
  none: string;
  more: string;
  less: string;
  expand: string;
  collapse: string;
  show: string;
  hide: string;
  type: string;
  value: string;
  key: string;
  description: string;
  time: string;
  date: string;
  ago: string;
  from: string;
  to: string;
  items: string;
  page: string;
  of: string;
  perPage: string;
  first: string;
  last: string;
  loadMore: string;
  noMore: string;
  retry: string;
  loadFailed: string;
  noCluster: string;
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
  healthy: string;
  unhealthy: string;
  degraded: string;
  active: string;
  inactive: string;
  online: string;
  offline: string;
  connected: string;
  disconnected: string;
  terminated: string;
  waiting: string;
  creating: string;
  deleting: string;
  updating: string;
  scaling: string;
  error: string;
  warning: string;
  info: string;
  critical: string;
}

// 审计翻译
export interface AuditTranslations {
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
  lastHour: string;
  last24Hours: string;
  last7Days: string;
  allTime: string;
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
  // 角色标签
  roles: {
    guest: string;
    viewer: string;
    operator: string;
    admin: string;
  };
  // 资源类型标签
  resources: {
    user: string;
    pod: string;
    deployment: string;
    node: string;
    configmap: string;
    secret: string;
    command: string;
    notify: string;
  };
  // 操作+资源组合标签
  actionLabels: {
    loginUser: string;
    executePod: string;
    executeDeployment: string;
    executeNode: string;
    executeCommand: string;
    readPod: string;
    readConfigmap: string;
    readSecret: string;
    createUser: string;
    updateUser: string;
    updateNotify: string;
    deleteUser: string;
  };
  // 操作名称（回退用）
  actionNames: {
    execute: string;
    read: string;
    create: string;
    update: string;
    delete: string;
    login: string;
  };
}

// Pod 页面翻译
export interface PodTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allNamespaces: string;
  allNodes: string;
  allStatus: string;
  viewDetails: string;
  viewLogs: string;
  restart: string;
  restartConfirmTitle: string;
  restartConfirmMessage: string;
  containers: string;
  container: string;
  image: string;
  ports: string;
  resources: string;
  limits: string;
  requests: string;
  volumeMounts: string;
  envVars: string;
  conditions: string;
  events: string;
  labels: string;
  annotations: string;
  ownerReferences: string;
  restarts: string;
  age: string;
  node: string;
  ip: string;
  hostIP: string;
  qosClass: string;
  serviceAccount: string;
  phase: string;
  reason: string;
  message: string;
  startTime: string;
  lastState: string;
  currentState: string;
  ready: string;
  started: string;
  restartCount: string;
  logs: string;
  logsTitle: string;
  logsLoading: string;
  logsEmpty: string;
  logsError: string;
  tailLines: string;
  follow: string;
  timestamps: string;
  previous: string;
  selectContainer: string;
  noContainers: string;
  // Detail modal tabs and sections
  overview: string;
  network: string;
  scheduling: string;
  basicInfo: string;
  resourceUsage: string;
  noVolumes: string;
  lastTerminatedReason: string;
  restartTimes: string;
  controller: string;
}

// Node 页面翻译
export interface NodeTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allStatus: string;
  viewDetails: string;
  cordon: string;
  uncordon: string;
  cordonConfirmTitle: string;
  cordonConfirmMessage: string;
  uncordonConfirmTitle: string;
  uncordonConfirmMessage: string;
  schedulable: string;
  unschedulable: string;
  labels: string;
  annotations: string;
  taints: string;
  conditions: string;
  capacity: string;
  allocatable: string;
  addresses: string;
  nodeInfo: string;
  kubeletVersion: string;
  osImage: string;
  containerRuntime: string;
  kernelVersion: string;
  architecture: string;
  pods: string;
  podsOnNode: string;
  metrics: string;
  cpuUsage: string;
  memoryUsage: string;
  diskUsage: string;
  podCount: string;
  role: string;
  version: string;
  internalIP: string;
  externalIP: string;
  hostname: string;
  // Detail modal
  overview: string;
  resources: string;
  basicInfo: string;
  resourceUsage: string;
  pressureWarning: string;
  memoryPressure: string;
  diskPressure: string;
  pidPressure: string;
  networkUnavailable: string;
  cordoned: string;
  cpuCapacity: string;
  cpuAllocatable: string;
  memoryCapacity: string;
  memoryAllocatable: string;
  podCapacity: string;
  podAllocatable: string;
  ephemeralStorage: string;
  capacityAllocatable: string;
  noConditions: string;
  noTaints: string;
  noLabels: string;
}

// Deployment 页面翻译
export interface DeploymentTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allNamespaces: string;
  allStatus: string;
  viewDetails: string;
  scale: string;
  restart: string;
  updateImage: string;
  scaleConfirmTitle: string;
  scaleConfirmMessage: string;
  restartConfirmTitle: string;
  restartConfirmMessage: string;
  updateImageTitle: string;
  updateImageMessage: string;
  replicas: string;
  readyReplicas: string;
  availableReplicas: string;
  unavailableReplicas: string;
  updatedReplicas: string;
  strategy: string;
  selector: string;
  labels: string;
  annotations: string;
  conditions: string;
  pods: string;
  template: string;
  containers: string;
  currentImage: string;
  newImage: string;
  desiredReplicas: string;
  currentReplicas: string;
  age: string;
  revision: string;
  // Detail modal tabs and sections
  overview: string;
  scheduling: string;
  replicaSets: string;
  basicInfo: string;
  adjustReplicas: string;
  updateStrategy: string;
  otherConfig: string;
  schedulingConfig: string;
  desired: string;
  ready: string;
  available: string;
  updated: string;
  paused: string;
  current: string;
  noContainers: string;
  noReplicaSets: string;
  noLabels: string;
  noAnnotations: string;
  image: string;
  ports: string;
  probes: string;
  envVars: string;
  volumeMounts: string;
  loadFailed: string;
  confirmUpdateImage: string;
  confirmUpdateImageMessage: string;
  confirmScale: string;
  confirmScaleMessage: string;
  hostNetwork: string;
  strategyType: string;
}

// Service 页面翻译
export interface ServiceTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allNamespaces: string;
  allTypes: string;
  viewDetails: string;
  serviceType: string;
  clusterIP: string;
  externalIP: string;
  ports: string;
  port: string;
  targetPort: string;
  nodePort: string;
  protocol: string;
  selector: string;
  labels: string;
  annotations: string;
  endpoints: string;
  noEndpoints: string;
  sessionAffinity: string;
  externalTrafficPolicy: string;
  loadBalancerIP: string;
  age: string;
  // Detail modal tabs and sections
  overview: string;
  basicInfo: string;
  loadFailed: string;
  noPorts: string;
  noSelector: string;
  endpointStatus: string;
  ready: string;
  notReady: string;
  total: string;
  trafficPolicy: string;
}

// Namespace 页面翻译
export interface NamespaceTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allStatus: string;
  viewDetails: string;
  labels: string;
  annotations: string;
  resourceQuotas: string;
  limitRanges: string;
  configMaps: string;
  secrets: string;
  pods: string;
  services: string;
  deployments: string;
  age: string;
  phase: string;
  finalizers: string;
  noConfigMaps: string;
  noSecrets: string;
  viewData: string;
  dataTitle: string;
  entries: string;
  secretType: string;
  masked: string;
  reveal: string;
  // Detail modal tabs and sections
  overview: string;
  quotas: string;
  basicInfo: string;
  loadFailed: string;
  podStatus: string;
  total: string;
  workloads: string;
  network: string;
  config: string;
  resourceUsage: string;
  utilization: string;
  noQuotas: string;
  noLabels: string;
  noAnnotations: string;
  noData: string;
}

// Ingress 页面翻译
export interface IngressTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allNamespaces: string;
  viewDetails: string;
  ingressClass: string;
  rules: string;
  host: string;
  path: string;
  pathType: string;
  backend: string;
  serviceName: string;
  servicePort: string;
  tls: string;
  tlsHosts: string;
  secretName: string;
  labels: string;
  annotations: string;
  defaultBackend: string;
  noRules: string;
  age: string;
  // Detail modal
  overview: string;
  routingRules: string;
  basicInfo: string;
  loadFailed: string;
  tlsStatus: string;
  tlsEnabled: string;
  tlsDisabled: string;
  ruleStatistics: string;
  pathCount: string;
  tlsCertificates: string;
  noTlsConfig: string;
  noAnnotations: string;
}

// Alert 页面翻译
export interface AlertTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allSeverities: string;
  allTypes: string;
  viewDetails: string;
  severity: string;
  type: string;
  source: string;
  message: string;
  timestamp: string;
  acknowledged: string;
  acknowledge: string;
  resolve: string;
  silence: string;
  labels: string;
  annotations: string;
  noAlerts: string;
  critical: string;
  warning: string;
  info: string;
  // 告警列表页面
  loadFailed: string;
  refresh: string;
  aiAnalyze: string;
  noAlertsTitle: string;
  noAlertsDescription: string;
  time: string;
  level: string;
  resourceType: string;
  namespace: string;
  resourceName: string;
  reason: string;
  totalAlerts: string;
  selectedCount: string;
  selectAll: string;
  analyzeHint: string;
}

// Overview 页面翻译
export interface OverviewTranslations {
  clusterHealth: string;
  nodeReady: string;
  podHealthy: string;
  nodes: string;
  clusterAvgCpu: string;
  clusterAvgMem: string;
  alerts: string;
  nodeResourceUsage: string;
  noNodeData: string;
  recentAlerts: string;
  noRecentAlerts: string;
  workloadSummary: string;
  podStatus: string;
  cpuPeak: string;
  memPeak: string;
  alertDetails: string;
  time: string;
  kind: string;
  reason: string;
  deploymentsLabel: string;
  daemonSetsLabel: string;
  statefulSetsLabel: string;
  jobsLabel: string;
  run: string;
  done: string;
  fail: string;
}

// Workbench 页面翻译
export interface WorkbenchTranslations {
  pageDescription: string;
  recentEvents: string;
  quickActions: string;
  noEvents: string;
  eventType: string;
  eventMessage: string;
  eventTime: string;
  podOperations: string;
  nodeOperations: string;
  deploymentOperations: string;
  // 快捷入口描述
  enter: string;
  aiDescription: string;
  clustersDescription: string;
  agentsDescription: string;
  notificationsDescription: string;
  // 开发中
  comingSoon: string;
  comingSoonDesc: string;
}

// Users 页面翻译
export interface UsersTranslations {
  pageDescription: string;
  addUser: string;
  editUser: string;
  deleteUser: string;
  deleteConfirmTitle: string;
  deleteConfirmMessage: string;
  username: string;
  displayName: string;
  email: string;
  role: string;
  status: string;
  lastLogin: string;
  lastLoginIP: string;
  createdAt: string;
  roleAdmin: string;
  roleOperator: string;
  roleViewer: string;
  statusActive: string;
  statusDisabled: string;
  passwordPlaceholder: string;
  changeRole: string;
  changeStatus: string;
  enable: string;
  disable: string;
  cannotDeleteAdmin: string;
  cannotDisableAdmin: string;
}

// Roles 页面翻译
export interface RolesTranslations {
  pageDescription: string;
  // 角色描述
  adminDescription: string;
  operatorDescription: string;
  viewerDescription: string;
  // 权限矩阵
  permissionMatrix: string;
  permissionMatrixDescription: string;
  resource: string;
  notes: string;
  // 分类
  categorySystem: string;
  categoryCluster: string;
  categoryMonitoring: string;
  categoryAI: string;
  // 资源
  userManagement: string;
  roleAssignment: string;
  auditLogs: string;
  notificationConfig: string;
  aiConfig: string;
  metricsView: string;
  logsView: string;
  alertRules: string;
  aiDiagnosis: string;
  aiWorkbench: string;
  // 权限标签
  permissionFull: string;
  permissionReadOnly: string;
  permissionPartial: string;
  permissionNone: string;
  // 权限说明
  permissionLevelDescription: string;
  fullPermission: string;
  fullPermissionDesc: string;
  readOnlyPermission: string;
  readOnlyPermissionDesc: string;
  partialPermission: string;
  partialPermissionDesc: string;
  noPermission: string;
  noPermissionDesc: string;
  // 备注
  noteViewUserList: string;
  noteOperatorSilenceAlert: string;
}

// Clusters 页面翻译
export interface ClustersTranslations {
  pageDescription: string;
  addCluster: string;
  clusterName: string;
  clusterID: string;
  status: string;
  nodeCount: string;
  podCount: string;
  version: string;
  lastSync: string;
  connected: string;
  disconnected: string;
  noClusters: string;
}

// Agents 页面翻译
export interface AgentsTranslations {
  pageDescription: string;
  agentName: string;
  clusterID: string;
  status: string;
  version: string;
  lastHeartbeat: string;
  uptime: string;
  connected: string;
  disconnected: string;
  noAgents: string;
}

// Notifications 页面翻译
export interface NotificationsTranslations {
  pageDescription: string;
  loadFailed: string;
  // Slack
  slackNotify: string;
  slackConfig: string;
  webhookUrl: string;
  webhookUrlHint: string;
  channel: string;
  // Email
  emailNotify: string;
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
  // 状态
  statusEnabled: string;
  statusIncomplete: string;
  statusDisabled: string;
  configIncomplete: string;
  // 操作
  enabled: string;
  test: string;
  save: string;
  testMessage: string;
  testSuccess: string;
  testFailed: string;
  saveSuccess: string;
  saveFailed: string;
  slackSaved: string;
  emailSaved: string;
}

// Login 页面翻译
export interface LoginTranslations {
  title: string;
  subtitle: string;
  usernamePlaceholder: string;
  passwordPlaceholder: string;
  loginButton: string;
  loggingIn: string;
  loginSuccess: string;
  loginFailed: string;
  invalidCredentials: string;
  sessionExpired: string;
  pleaseLogin: string;
}

// Confirm Dialog 翻译
export interface ConfirmTranslations {
  defaultTitle: string;
  defaultMessage: string;
  confirmButton: string;
  cancelButton: string;
  deleteTitle: string;
  deleteMessage: string;
  warningTitle: string;
  dangerTitle: string;
}

// Table 翻译
export interface TableTranslations {
  noData: string;
  loading: string;
  error: string;
  showing: string;
  entries: string;
  pageOf: string;
  rowsPerPage: string;
  firstPage: string;
  previousPage: string;
  nextPage: string;
  lastPage: string;
  sortAsc: string;
  sortDesc: string;
  filterPlaceholder: string;
}

// DaemonSet 翻译
export interface DaemonSetTranslations {
  pageDescription: string;
  desiredScheduled: string;
  currentScheduled: string;
  numberReady: string;
  numberAvailable: string;
  numberMisscheduled: string;
  updateStrategy: string;
  selector: string;
  labels: string;
  annotations: string;
  conditions: string;
  pods: string;
  template: string;
  containers: string;
  nodeSelector: string;
  tolerations: string;
  age: string;
}

// StatefulSet 翻译
export interface StatefulSetTranslations {
  pageDescription: string;
  replicas: string;
  readyReplicas: string;
  currentReplicas: string;
  updatedReplicas: string;
  updateStrategy: string;
  serviceName: string;
  podManagementPolicy: string;
  selector: string;
  labels: string;
  annotations: string;
  conditions: string;
  volumeClaimTemplates: string;
  pods: string;
  template: string;
  containers: string;
  age: string;
  revision: string;
}

// Placeholder 页面翻译
export interface PlaceholderTranslations {
  developingTitle: string;
  developingMessage: string;
}

// SLO 页面翻译
export interface SLOTranslations {
  // 页面标题
  pageTitle: string;
  pageDescription: string;
  noData: string;
  noDataHint: string;
  refreshing: string;
  lastUpdated: string;
  loadFailed: string;
  noCluster: string;
  // 状态
  healthy: string;
  warning: string;
  critical: string;
  degraded: string;
  atRisk: string;
  breached: string;
  unknown: string;
  // 趋势
  trendUp: string;
  trendDown: string;
  trendStable: string;
  // 时间范围
  day: string;
  week: string;
  month: string;
  selectPeriod: string;
  previousPeriod: string;
  currentVsPrevious: string;
  // 指标
  availability: string;
  p95Latency: string;
  p99Latency: string;
  latency: string;
  errorRate: string;
  rps: string;
  throughput: string;
  totalRequests: string;
  successRequests: string;
  errorRequests: string;
  avgThroughput: string;
  // 详情弹窗
  domainDetail: string;
  targetDomain: string;
  currentMetrics: string;
  sloTarget: string;
  sloStatus: string;
  sloAchievement: string;
  errorBudget: string;
  errorBudgetDetail: string;
  remainingBudget: string;
  allowedErrors: string;
  actualErrors: string;
  remainingQuota: string;
  trafficStats: string;
  history: string;
  noTarget: string;
  setTarget: string;
  editTarget: string;
  deleteTarget: string;
  configTarget: string;
  configSloTarget: string;
  targetAvailability: string;
  targetAvailabilityHint: string;
  targetP95: string;
  targetP95Hint: string;
  errorRateThreshold: string;
  errorRateAutoCalc: string;
  remaining: string;
  consumed: string;
  actual: string;
  target: string;
  threshold: string;
  achieved: string;
  notAchieved: string;
  exceeded: string;
  // Tabs
  tabSloStatus: string;
  tabServices: string;
  tabCompare: string;
  tabOverview: string;
  tabMesh: string;
  tabLatency: string;
  // 服务网格
  serviceTopology: string;
  meshOverview: string;
  mtls: string;
  inbound: string;
  outbound: string;
  noCallData: string;
  callRelation: string;
  p50Latency: string;
  avgLatency: string;
  // 趋势图
  sloTrend: string;
  errorBudgetBurn: string;
  current: string;
  estimatedExhaust: string;
  // 服务相关
  services: string;
  servicesCount: string;
  backendServices: string;
  noServiceData: string;
  totalBackendServices: string;
  // 汇总卡片
  totalServices: string;
  linkerdMesh: string;
  monitoredDomains: string;
  avgAvailability: string;
  avgP95: string;
  totalRPS: string;
  alertCount: string;
  severe: string;
  domainSloStatus: string;
  // 图表
  hourlyTrend: string;
  dailyTrend: string;
  // 单位
  ms: string;
  percent: string;
  reqPerSec: string;
  // 操作
  viewDetail: string;
  save: string;
  saving: string;
  cancel: string;
  saveFailed: string;
  // 延迟分布 / 服务详情
  latencyDistribution: string;
  methodBreakdown: string;
  statusCodeBreakdown: string;
  clearSelection: string;
  requests: string;
  loading: string;
  // 说明
  dataSourceTitle: string;
  dataSourceDesc: string;
}

// Commands 页面翻译
export interface CommandsTranslations {
  pageDescription: string;
  searchPlaceholder: string;
  allSources: string;
  allStatus: string;
  allActions: string;
  source: string;
  target: string;
  params: string;
  result: string;
  duration: string;
  noCommands: string;
  viewDetails: string;
  commandId: string;
  errorMessage: string;
  createdAt: string;
  startedAt: string;
  finishedAt: string;
  sources: {
    web: string;
    ai: string;
  };
  statuses: {
    pending: string;
    running: string;
    success: string;
    failed: string;
    timeout: string;
  };
  actions: {
    restart: string;
    scale: string;
    delete_pod: string;
    cordon: string;
    uncordon: string;
    update_image: string;
  };
}

// AI Settings 页面翻译
export interface AISettingsPageTranslations {
  // 页面标题
  pageTitle: string;
  pageDescription: string;
  // 全局设置
  globalSettings: string;
  globalSettingsDesc: string;
  aiFeature: string;
  enableAI: string;
  enableAIDesc: string;
  enabled: string;
  disabled: string;
  toolTimeout: string;
  toolTimeoutDesc: string;
  seconds: string;
  saveTimeout: string;
  // 提供商列表
  providerList: string;
  addProvider: string;
  addFirstProvider: string;
  noProviders: string;
  // 提供商卡片
  active: string;
  activate: string;
  settings: string;
  edit: string;
  delete: string;
  model: string;
  apiKeyConfigured: string;
  // 提供商弹窗
  newProvider: string;
  editProvider: string;
  addProviderTitle: string;
  editProviderTitle: string;
  name: string;
  namePlaceholder: string;
  providerName: string;
  providerNamePlaceholder: string;
  provider: string;
  providerType: string;
  selectProvider: string;
  apiKey: string;
  apiKeyPlaceholder: string;
  apiKeyUpdatePlaceholder: string;
  apiKeyHint: string;
  current: string;
  selectModel: string;
  customModel: string;
  customModelPlaceholder: string;
  description: string;
  descriptionPlaceholder: string;
  cancel: string;
  save: string;
  saving: string;
  // Toast 消息
  loadFailed: string;
  requiredFields: string;
  apiKeyRequired: string;
  providerUpdated: string;
  providerAdded: string;
  saveFailed: string;
  cannotDeleteActive: string;
  confirmDelete: string;
  providerDeleted: string;
  deleteFailed: string;
  providerActivated: string;
  activateFailed: string;
  aiEnabled: string;
  aiDisabled: string;
  settingChangeFailed: string;
  timeoutSaved: string;
}

// AI Chat 页面翻译
export interface AIChatPageTranslations {
  // 状态页面
  loading: string;
  checkingConfig: string;
  notEnabled: string;
  notEnabledDesc: string;
  notConfigured: string;
  notConfiguredDesc: string;
  goToSettings: string;
  // 演示模式
  demoMode: string;
  demoModeDesc: string;
  login: string;
  // 告警分析
  alertAnalysis: string;
  // ChatPanel
  conversationHistory: string;
  executionDetails: string;
  aiAssistant: string;
  aiAssistantDesc: string;
  // 快捷问题
  quickQuestions: {
    podRestart: string;
    nodeResource: string;
    alertEvents: string;
    deploymentStatus: string;
  };
  // ChatInput
  chatInput: {
    placeholderStreaming: string;
    placeholderNormal: string;
    stopButton: string;
    sendButton: string;
    disclaimer: string;
  };
  // InspectorPanel
  inspector: {
    title: string;
    clusterContext: string;
    clusterId: string;
    notSelected: string;
    status: string;
    querying: string;
    connected: string;
    currentQuestion: string;
    thinkingRounds: string;
    inProgress: string;
    executedCommands: string;
    commandsUnit: string;
    roundsUnit: string;
    inputTokens: string;
    outputTokens: string;
    totalTokens: string;
    waitingQuestion: string;
    conversationOverview: string;
    conversationRounds: string;
    totalCommands: string;
    totalTokensLabel: string;
    aiCapabilities: string;
    canDo: string;
    cannotDo: string;
    capQueryResources: string;
    capViewLogs: string;
    capViewEvents: string;
    capFilterByLabel: string;
    capNoWrite: string;
    capNoSecrets: string;
    capNoSystemNs: string;
    capNoSensitive: string;
  };
  // ConversationSidebar
  sidebar: {
    title: string;
    newConversation: string;
    noConversations: string;
    deleteConversation: string;
    today: string;
    yesterday: string;
    past7Days: string;
    past30Days: string;
    older: string;
  };
}

// 完整翻译结构
export interface Translations {
  locale: "zh" | "ja";  // 语言标识符，用于日期格式化等
  nav: NavTranslations;
  common: CommonTranslations;
  status: StatusTranslations;
  audit: AuditTranslations;
  pod: PodTranslations;
  node: NodeTranslations;
  deployment: DeploymentTranslations;
  service: ServiceTranslations;
  namespace: NamespaceTranslations;
  ingress: IngressTranslations;
  alert: AlertTranslations;
  overview: OverviewTranslations;
  workbench: WorkbenchTranslations;
  users: UsersTranslations;
  roles: RolesTranslations;
  clusters: ClustersTranslations;
  agents: AgentsTranslations;
  notifications: NotificationsTranslations;
  login: LoginTranslations;
  confirm: ConfirmTranslations;
  table: TableTranslations;
  daemonset: DaemonSetTranslations;
  statefulset: StatefulSetTranslations;
  placeholder: PlaceholderTranslations;
  commands: CommandsTranslations;
  slo: SLOTranslations;
  aiSettingsPage: AISettingsPageTranslations;
  aiChatPage: AIChatPageTranslations;
}

// 国际化上下文
export interface I18nContextType {
  language: Language;
  setLanguage: (lang: Language) => void;
  t: Translations;
}
