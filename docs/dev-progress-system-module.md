# System 模块开发进度

> 最后更新: 2026-01-04
> **状态: Phase 1-3 全部完成**

## 模块架构

```
System（系统）
├── 📊 Monitoring（监控）
│   ├── Metrics      ✅ 已完成
│   ├── Logs         ✅ Phase 3 完成
│   └── Alerts       ✅ Phase 3 完成
│
├── 👥 Access Control（访问控制）
│   ├── Users        ✅ Phase 1 完成
│   ├── Roles        ✅ Phase 2 完成
│   └── Audit        ✅ Phase 1 完成
│
└── ⚙️ Settings（配置）
    ├── Clusters     ✅ Phase 2 完成
    ├── Agents       ✅ Phase 2 完成
    └── Notifications ✅ Phase 1 完成
```

## Phase 1: 真实化现有页面 ✅ 已完成

### 1.1 Users 页面真实化
- **状态**: ✅ 已完成
- **路由**: `/system/users`
- **文件**: `atlhyper_web/src/app/system/users/page.tsx`
- **后端 API**:
  - `GET /uiapi/auth/user/list` - 获取用户列表 ✅
  - `POST /uiapi/auth/user/register` - 注册用户 ✅ (需 Admin)
  - `POST /uiapi/auth/user/update-role` - 更新角色 ✅ (需 Admin)
- **功能**:
  - [x] 对接真实用户列表 API
  - [x] 添加用户弹窗（仅 Admin 可见）
  - [x] 编辑用户角色弹窗

### 1.2 Audit 页面真实化
- **状态**: ✅ 已完成
- **路由**: `/system/audit`
- **文件**: `atlhyper_web/src/app/system/audit/page.tsx`
- **后端 API**:
  - `GET /uiapi/auth/userauditlogs/list` - 获取审计日志 ✅
- **功能**:
  - [x] 对接真实审计日志 API
  - [x] 添加时间范围过滤
  - [x] 添加用户/操作类型过滤

### 1.3 Notifications 页面
- **状态**: ✅ 已完成
- **路由**: `/system/notifications`
- **文件**: `atlhyper_web/src/app/system/notifications/page.tsx`
- **后端 API**:
  - `POST /uiapi/config/slack/get` - 获取 Slack 配置 ✅
  - `POST /uiapi/config/slack/update` - 更新 Slack 配置 ✅ (需 Admin)
- **功能**:
  - [x] Slack Webhook URL 配置表单
  - [x] 启用/禁用开关
  - [x] 发送间隔配置

## Phase 2: 新增核心页面 ✅ 已完成

### 2.1 Roles 页面
- **状态**: ✅ 已完成
- **路由**: `/system/roles`
- **文件**: `atlhyper_web/src/app/system/roles/page.tsx`
- **后端 API**: 无需（前端静态展示）
- **功能**:
  - [x] 角色卡片展示（Admin/Operator/Viewer）
  - [x] 权限矩阵表格
  - [x] 分类展示（系统/集群/监控/AI）
  - [x] 权限级别说明

### 2.2 Agents 页面
- **状态**: ✅ 已完成
- **路由**: `/system/agents`
- **文件**: `atlhyper_web/src/app/system/agents/page.tsx`
- **后端 API**: 使用现有 Node API
  - `POST /uiapi/node/overview` - 获取节点信息
  - `POST /uiapi/cluster/overview` - 获取资源使用率
- **功能**:
  - [x] Agent 卡片式列表
  - [x] 在线/离线状态展示
  - [x] CPU/内存使用率显示
  - [x] 节点信息（IP、OS、规格）
  - [x] 统计卡片（总数/在线/离线/异常）
  - [x] 自动刷新（30秒）

### 2.3 Clusters 页面
- **状态**: ✅ 已完成
- **路由**: `/system/clusters`
- **文件**: `atlhyper_web/src/app/system/clusters/page.tsx`
- **后端 API**: 使用现有 Overview API
  - `POST /uiapi/cluster/overview` - 获取集群概览
- **功能**:
  - [x] 集群卡片展示
  - [x] 健康状态指示
  - [x] 资源统计（节点/Pod/CPU/内存）
  - [x] 快速操作链接

## Phase 3: 高级功能 ✅ 已完成

### 3.1 Logs 页面增强
- **状态**: ✅ 已完成
- **路由**: `/system/logs`
- **文件**: `atlhyper_web/src/app/system/logs/page.tsx`
- **后端 API**:
  - `POST /uiapi/event/logs` - 获取事件日志 ✅
- **功能**:
  - [x] 对接真实事件日志 API
  - [x] 统计卡片（总事件/Error/Warning/Info/资源类型）
  - [x] 高级过滤（级别/Kind/Namespace）
  - [x] 时间范围选择（1/3/7/14/30天）
  - [x] 全文搜索
  - [x] 自动刷新开关
  - [x] 导出功能（JSON/CSV）

### 3.2 Alerts 页面
- **状态**: ✅ 已完成
- **路由**: `/system/alerts`
- **文件**: `atlhyper_web/src/app/system/alerts/page.tsx`
- **后端 API**: 使用 Event API
  - `POST /uiapi/event/logs` - 获取告警历史
- **功能**:
  - [x] 告警规则展示（预设规则）
  - [x] 规则启用/禁用切换
  - [x] 告警历史 Tab
  - [x] 统计卡片（规则数/已启用/严重/警告）
  - [x] 搜索和级别过滤
  - [ ] 规则 CRUD（待后端 API）

## 导航结构 ✅ 已更新

`Sidebar.tsx` 当前配置:
```typescript
{
  key: "system",
  icon: Activity,
  children: [
    { key: "metrics", href: "/system/metrics" },
    { key: "logs", href: "/system/logs" },
    { key: "alerts", href: "/system/alerts" },
    { key: "users", href: "/system/users" },
    { key: "roles", href: "/system/roles" },
    { key: "audit", href: "/system/audit" },
    { key: "clusters", href: "/system/clusters" },
    { key: "agents", href: "/system/agents" },
    { key: "notifications", href: "/system/notifications" },
  ],
}
```

## 相关文件路径

```
atlhyper_web/
├── src/
│   ├── app/system/
│   │   ├── metrics/page.tsx     ✅ 完成
│   │   ├── logs/page.tsx        ✅ 真实 API + 高级过滤
│   │   ├── alerts/page.tsx      ✅ 规则展示 + 历史
│   │   ├── users/page.tsx       ✅ 真实 API
│   │   ├── roles/page.tsx       ✅ 权限矩阵
│   │   ├── audit/page.tsx       ✅ 真实 API
│   │   ├── clusters/page.tsx    ✅ 真实 API
│   │   ├── agents/page.tsx      ✅ 真实 API
│   │   └── notifications/page.tsx ✅ 真实 API
│   ├── api/
│   │   ├── auth.ts              用户认证 API
│   │   ├── node.ts              节点 API
│   │   ├── overview.ts          概览 API
│   │   ├── event.ts             事件日志 API
│   │   └── config.ts            配置 API
│   ├── types/
│   │   ├── auth.ts              用户类型定义
│   │   ├── cluster.ts           集群类型定义
│   │   └── i18n.ts              国际化类型
│   ├── i18n/locales/
│   │   ├── zh.ts                中文翻译
│   │   └── ja.ts                日文翻译
│   └── components/navigation/
│       └── Sidebar.tsx          导航栏
```

## 后端 API 清单

### 已对接 API
| 路由 | 方法 | 说明 | 页面 |
|------|------|------|------|
| `/uiapi/auth/user/list` | GET | 用户列表 | Users |
| `/uiapi/auth/userauditlogs/list` | GET | 审计日志 | Audit |
| `/uiapi/auth/user/register` | POST | 注册用户 | Users |
| `/uiapi/auth/user/update-role` | POST | 更新角色 | Users |
| `/uiapi/config/slack/get` | POST | Slack 配置 | Notifications |
| `/uiapi/config/slack/update` | POST | 更新配置 | Notifications |
| `/uiapi/node/overview` | POST | 节点概览 | Agents |
| `/uiapi/cluster/overview` | POST | 集群概览 | Clusters |
| `/uiapi/event/logs` | POST | 事件日志 | Logs, Alerts |

### 待开发 API
| 路由 | 方法 | 说明 | 备注 |
|------|------|------|------|
| `/uiapi/alerts/rules` | CRUD | 告警规则 | 当前使用预设规则 |

## 路由总览（21 个页面）

```
/                        首页重定向
/overview               ✅ 集群概览
/workbench              ✅ AI 工作台
/cluster/pod            ✅ Pod 管理
/cluster/node           ✅ Node 管理
/cluster/deployment     ✅ Deployment 管理
/cluster/service        ✅ Service 管理
/cluster/namespace      ✅ Namespace 管理
/cluster/ingress        ✅ Ingress 管理
/cluster/alert          ✅ Alert 管理
/system/metrics         ✅ 系统指标
/system/logs            ✅ 事件日志（增强版）
/system/alerts          ✅ 告警管理
/system/users           ✅ 用户管理
/system/roles           ✅ 角色权限
/system/audit           ✅ 审计日志
/system/clusters        ✅ 集群管理
/system/agents          ✅ Agent 管理
/system/notifications   ✅ 通知配置
```

## 用户认证系统

### 认证架构
- **JWT 认证**: HS256 签名，24 小时有效期
- **密码加密**: bcrypt 哈希
- **Token 管理**: 前端 localStorage 存储

### 角色权限
| 角色 | 值 | 权限范围 |
|------|-----|----------|
| Viewer | 1 | 无需登录，只读查看 |
| Operator | 2 | 需登录，可执行操作 |
| Admin | 3 | 需登录，用户管理 |

### 配置项（环境变量）
```bash
# JWT 配置
MASTER_JWT_SECRET_KEY=atlhyper_jwt_secret_key_change_in_production  # 签名密钥
MASTER_JWT_TOKEN_EXPIRY=24h                                         # Token 有效期
MASTER_JWT_MIN_PASSWORD_LEN=6                                       # 密码最小长度
```

### Token 过期处理
1. 后端返回 401（Token 过期）
2. 前端拦截器捕获 → 弹出登录对话框
3. 用户重新登录获取新 Token

### 相关文件
```
atlhyper_master/
├── config/
│   ├── types.go      # JWTConfig 结构体
│   ├── defaults.go   # JWT 默认值
│   └── loader.go     # 配置加载
├── server/api/auth/
│   ├── jwt.go        # Token 生成/解析
│   ├── middleware.go # 认证中间件
│   └── handler.go    # 登录/注册处理
└── db/repository/user/
    └── registeruser.go # 用户注册（含密码验证）

atlhyper_web/src/
├── api/request.ts      # 401/403 拦截
├── hooks/useAuthError.ts # 权限错误处理
└── store/authStore.ts  # 认证状态管理
```

## 开发总结

System 模块共实现 9 个页面：
- **Phase 1**: Users, Audit, Notifications（真实 API 对接）
- **Phase 2**: Roles, Agents, Clusters（新增功能页面）
- **Phase 3**: Logs 增强, Alerts（高级功能）

所有页面均支持：
- 响应式布局
- 暗色主题
- 国际化（中/日）
- 加载状态
- 错误处理
