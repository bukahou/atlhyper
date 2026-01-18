# AtlHyper Web 国际化任务进度

> 最后更新: 2026-01-18 (已完成)

---

## 概述

将 AtlHyper Web 前端进行完整的国际化（中文 + 日语）。范围包括：
- 所有页面（主页面、详情页）
- 所有组件（Modal 弹窗、详情卡片、表格、表单）
- 所有交互文案（按钮、提示、确认框、Toast）

---

## 当前状态

### ✅ 国际化已完成

国际化框架和所有页面/组件已完成：
- `src/i18n/index.ts` - 国际化入口
- `src/i18n/context.tsx` - React Context Provider
- `src/i18n/locales/zh.ts` - 中文翻译（完整）
- `src/i18n/locales/ja.ts` - 日语翻译（完整）
- `src/types/i18n.ts` - 类型定义（完整）

### 已翻译内容
- ✅ `nav` - 导航菜单
- ✅ `common` - 通用文案
- ✅ `status` - 状态标签
- ✅ `audit` - 审计页面
- ✅ `pod` - Pod 模块
- ✅ `node` - Node 模块
- ✅ `deployment` - Deployment 模块
- ✅ `service` - Service 模块
- ✅ `namespace` - Namespace 模块
- ✅ `ingress` - Ingress 模块
- ✅ `alert` - Alert 模块
- ✅ `overview` - 概览页
- ✅ `workbench` - 工作台
- ✅ `users` - 用户管理
- ✅ `clusters` - 集群管理
- ✅ `agents` - Agent 管理
- ✅ `notifications` - 通知配置
- ✅ `login` - 登录
- ✅ `confirm` - 确认对话框
- ✅ `table` - 数据表格
- ✅ `daemonset` - DaemonSet
- ✅ `statefulset` - StatefulSet
- ✅ `placeholder` - 占位页面

---

## 任务列表

状态说明: ✅ 完成 | 🔄 进行中 | ⏳ 待开始

### 第一阶段：类型定义扩展

| 任务 | 状态 | 说明 |
|------|------|------|
| 扩展 `types/i18n.ts` 类型定义 | ✅ | 添加所有页面和组件所需的翻译 key |

### 第二阶段：页面国际化

#### Cluster 模块
| 页面 | 文件路径 | 状态 | 包含组件 |
|------|---------|------|---------|
| Pod 列表 | `app/cluster/pod/page.tsx` | ✅ | 表格、状态标签、操作按钮 |
| Pod 详情 | `components/pod/PodDetailModal.tsx` | ✅ | 详情卡片、Tab 标签、日志查看器 |
| Pod 日志 | `components/pod/PodLogsViewer.tsx` | ✅ | 日志面板、容器选择器 |
| Node 列表 | `app/cluster/node/page.tsx` | ✅ | 表格、状态标签、操作按钮 |
| Node 详情 | `components/node/NodeDetailModal.tsx` | ✅ | 详情卡片、指标展示 |
| Deployment 列表 | `app/cluster/deployment/page.tsx` | ✅ | 表格、状态标签、操作按钮 |
| Deployment 详情 | `components/deployment/DeploymentDetailModal.tsx` | ✅ | 详情卡片、副本信息、镜像信息 |
| Service 列表 | `app/cluster/service/page.tsx` | ✅ | 表格、类型标签 |
| Service 详情 | `components/service/ServiceDetailModal.tsx` | ✅ | 详情卡片、端口信息、Endpoint |
| Namespace 列表 | `app/cluster/namespace/page.tsx` | ✅ | 表格、状态标签 |
| Namespace 详情 | `components/namespace/NamespaceDetailModal.tsx` | ✅ | 详情卡片、ConfigMap/Secret Tab |
| Ingress 列表 | `app/cluster/ingress/page.tsx` | ✅ | 表格、路由规则 |
| Ingress 详情 | `components/ingress/IngressDetailModal.tsx` | ✅ | 详情卡片、规则列表 |
| Alert 列表 | `app/cluster/alert/page.tsx` | ✅ | 表格、级别标签 |
| DaemonSet 详情 | `components/daemonset/DaemonSetDetailModal.tsx` | ✅ | 详情卡片 |
| StatefulSet 详情 | `components/statefulset/StatefulSetDetailModal.tsx` | ✅ | 详情卡片 |

#### System 模块
| 页面 | 文件路径 | 状态 | 包含组件 |
|------|---------|------|---------|
| 概览 | `app/overview/page.tsx` | ✅ | 统计卡片、图表、资源列表 |
| 工作台 | `app/workbench/page.tsx` | ✅ | 占位页面 |
| 用户管理 | `app/system/users/page.tsx` | ✅ | 用户表格、添加用户弹窗、角色/状态操作 |
| 审计日志 | `app/system/audit/page.tsx` | ✅ | 已完成 |
| 集群管理 | `app/system/clusters/page.tsx` | ✅ | 占位页面 |
| Agent 管理 | `app/system/agents/page.tsx` | ✅ | 占位页面 |
| 通知配置 | `app/system/notifications/page.tsx` | ✅ | 占位页面 |
| 指标 | `app/system/metrics/page.tsx` | ✅ | 占位页面 |
| 日志 | `app/system/logs/page.tsx` | ✅ | 占位页面 |
| 告警管理 | `app/system/alerts/page.tsx` | ✅ | 占位页面 |
| 角色权限 | `app/system/roles/page.tsx` | ✅ | 占位页面 |

#### 通用组件
| 组件 | 文件路径 | 状态 | 说明 |
|------|---------|------|------|
| 登录对话框 | `components/auth/LoginDialog.tsx` | ✅ | 登录表单、错误提示 |
| 确认对话框 | `components/common/ConfirmDialog.tsx` | ✅ | 默认确认/取消按钮文案国际化 |
| 数据表格 | `components/common/DataTable.tsx` | ✅ | 分页文案国际化 |
| 页面头部 | `components/common/PageHeader.tsx` | ✅ | 刷新按钮 title |
| 统计卡片 | `components/common/StatsCard.tsx` | ✅ | 标签由调用方传入 |
| 状态徽章 | `components/common/StatusBadge.tsx` | ✅ | 状态由调用方传入 |
| Toast 提示 | `components/common/Toast.tsx` | ✅ | 消息由调用方传入 |
| 用户菜单 | `components/navigation/UserMenu.tsx` | ✅ | 菜单项国际化 |
| 语言切换器 | `components/navigation/LanguageSwitcher.tsx` | ✅ | 语言名称 |
| 集群选择器 | `components/navigation/ClusterSelector.tsx` | ✅ | 集群选择提示 |

### 第三阶段：翻译文件完善

| 任务 | 状态 | 说明 |
|------|------|------|
| 完善 `zh.ts` 中文翻译 | ✅ | 所有翻译 key 已添加 |
| 完善 `ja.ts` 日语翻译 | ✅ | 所有翻译 key 已添加 |

### 第四阶段：测试验证

| 任务 | 状态 | 说明 |
|------|------|------|
| TypeScript 编译检查 | ✅ | `npx tsc --noEmit` 无错误 |
| 中文模式全面测试 | ⏳ | 需手动测试 |
| 日语模式全面测试 | ⏳ | 需手动测试 |
| 语言切换测试 | ⏳ | 需手动测试 |

---

## 翻译 Key 规划

### 新增翻译模块

```typescript
// types/i18n.ts 计划扩展结构
interface Translations {
  nav: NavTranslations;        // 已有
  common: CommonTranslations;  // 已有，需扩展
  status: StatusTranslations;  // 已有，需扩展
  audit: AuditTranslations;    // 已有

  // 新增模块
  pod: PodTranslations;
  node: NodeTranslations;
  deployment: DeploymentTranslations;
  service: ServiceTranslations;
  namespace: NamespaceTranslations;
  ingress: IngressTranslations;
  configmap: ConfigMapTranslations;
  secret: SecretTranslations;
  overview: OverviewTranslations;
  workbench: WorkbenchTranslations;
  users: UsersTranslations;
  clusters: ClustersTranslations;
  agents: AgentsTranslations;
  notifications: NotificationsTranslations;
  login: LoginTranslations;
  confirm: ConfirmTranslations;
  table: TableTranslations;
}
```

---

## 工作流程

1. **单页面/组件处理** - 每次只处理一个页面或组件
2. **更新类型定义** - 先在 `types/i18n.ts` 添加新类型
3. **添加翻译** - 同时更新 `zh.ts` 和 `ja.ts`
4. **修改组件** - 替换硬编码文案为 `t.xxx`
5. **验证编译** - 确保 TypeScript 无错误

---

## 变更记录

| 日期 | 变更内容 |
|------|---------|
| 2026-01-18 | 创建国际化任务跟踪文档 |
| 2026-01-18 | 完成任务规划和范围分析 |
| 2026-01-18 | 完成 types/i18n.ts 类型定义扩展 |
| 2026-01-18 | 完成 zh.ts 和 ja.ts 翻译文件 |
| 2026-01-18 | 完成 Cluster 模块所有页面国际化（Pod, Node, Deployment, Service, Namespace, Ingress, Alert） |
| 2026-01-18 | 完成 System 模块所有页面国际化（Overview, Users, Audit, 占位页面等） |
| 2026-01-18 | 完成通用组件国际化（DataTable, ConfirmDialog） |
| 2026-01-18 | TypeScript 编译验证通过 |

---

## 注意事项

1. **不使用多线程** - 避免上下文快速耗尽
2. **逐个文件处理** - 一次只修改一个文件
3. **及时更新文档** - 每完成一项任务更新本文档
4. **TypeScript 类型安全** - 确保所有翻译 key 有类型定义
