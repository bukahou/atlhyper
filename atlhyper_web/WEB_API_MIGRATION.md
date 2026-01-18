# Web API 层迁移任务管理

> 目标：适配 Master V2 API
> 创建时间：2026-01-17
> 状态：**API 层已完成** ✅

---

## 一、响应格式变化

### 旧格式 (Old Master)
```json
{
  "code": 20000,
  "message": "success",
  "data": { ... }
}
```

### 新格式 (Master V2)
```json
// 成功响应 - 直接返回数据
{
  "message": "获取成功",
  "data": { ... }
}

// 或直接返回数据对象
{
  "clusters": [...],
  "total": 5
}

// 错误响应
{
  "error": "错误信息"
}
```

---

## 二、文件修改清单

### 2.1 核心文件

| 文件 | 状态 | 说明 |
|------|------|------|
| `src/api/request.ts` | ✅ 已完成 | 响应拦截器适配新格式，添加 put/del 方法 |
| `src/types/common.ts` | ✅ 已完成 | 更新请求参数类型 (snake_case) |

### 2.2 认证相关

| 文件 | 状态 | 说明 |
|------|------|------|
| `src/api/auth.ts` | ✅ 已完成 | 用户认证 API，路径 /api/v2/user/* |

### 2.3 资源查询

| 文件 | 状态 | 说明 |
|------|------|------|
| `src/api/cluster.ts` | ✅ 已完成 | 集群信息，GET /api/v2/clusters |
| `src/api/pod.ts` | ✅ 已完成 | Pod 查询和操作 |
| `src/api/node.ts` | ✅ 已完成 | Node 查询和操作 |
| `src/api/deployment.ts` | ✅ 已完成 | Deployment 查询和操作 |
| `src/api/service.ts` | ✅ 已完成 | Service 查询 |
| `src/api/ingress.ts` | ✅ 已完成 | Ingress 查询 |
| `src/api/namespace.ts` | ✅ 已完成 | Namespace 和 ConfigMap 查询 |
| `src/api/event.ts` | ✅ 已完成 | Event 查询 |

### 2.4 配置相关

| 文件 | 状态 | 说明 |
|------|------|------|
| `src/api/config.ts` | ✅ 已完成 | 通知配置 (Slack) |

---

## 三、API 路径映射详情

### 3.1 用户认证 (auth.ts)

```
POST /uiapi/user/login          → POST /api/v2/user/login
GET  /uiapi/user/list           → GET  /api/v2/user/list
POST /uiapi/user/register       → POST /api/v2/user/register
POST /uiapi/user/update-role    → POST /api/v2/user/update-role
POST /uiapi/user/delete         → POST /api/v2/user/delete
GET  /uiapi/system/audit/list   → GET  /api/v2/audit/logs
```

### 3.2 集群信息 (cluster.ts)

```
POST /uiapi/overview/cluster/list   → GET /api/v2/clusters
POST /uiapi/overview/cluster/detail → GET /api/v2/clusters/{id}
```

### 3.3 Pod (pod.ts)

```
# 查询
POST /uiapi/cluster/pod/list   → GET /api/v2/pods?cluster_id=xxx&namespace=xxx
POST /uiapi/cluster/pod/detail → GET /api/v2/pods/{uid}?cluster_id=xxx

# 操作
POST /uiapi/ops/pod/logs    → POST /api/v2/ops/pods/logs
POST /uiapi/ops/pod/restart → POST /api/v2/ops/pods/restart
```

### 3.4 Node (node.ts)

```
# 查询
POST /uiapi/cluster/node/list   → GET /api/v2/nodes?cluster_id=xxx
POST /uiapi/cluster/node/detail → GET /api/v2/nodes/{uid}?cluster_id=xxx

# 操作
POST /uiapi/ops/node/cordon   → POST /api/v2/ops/nodes/cordon
POST /uiapi/ops/node/uncordon → POST /api/v2/ops/nodes/uncordon
```

### 3.5 Deployment (deployment.ts)

```
# 查询
POST /uiapi/cluster/deployment/list   → GET /api/v2/deployments?cluster_id=xxx
POST /uiapi/cluster/deployment/detail → GET /api/v2/deployments/{uid}?cluster_id=xxx

# 操作
POST /uiapi/ops/workload/scale       → POST /api/v2/ops/deployments/scale
POST /uiapi/ops/deployment/restart   → POST /api/v2/ops/deployments/restart
POST /uiapi/ops/workload/updateImage → POST /api/v2/ops/deployments/image
```

### 3.6 其他资源

```
# Service
POST /uiapi/cluster/service/list   → GET /api/v2/services?cluster_id=xxx
POST /uiapi/cluster/service/detail → GET /api/v2/services/{uid}?cluster_id=xxx

# Ingress
POST /uiapi/cluster/ingress/list   → GET /api/v2/ingresses?cluster_id=xxx
POST /uiapi/cluster/ingress/detail → GET /api/v2/ingresses/{uid}?cluster_id=xxx

# Namespace
POST /uiapi/cluster/namespace/list   → GET /api/v2/namespaces?cluster_id=xxx
POST /uiapi/cluster/namespace/detail → GET /api/v2/namespaces/{uid}?cluster_id=xxx

# ConfigMap
POST /uiapi/cluster/configmap/detail → GET /api/v2/configmaps?cluster_id=xxx

# Event
POST /uiapi/event/list → GET /api/v2/events?cluster_id=xxx
```

### 3.7 通知配置 (config.ts)

```
POST /uiapi/system/notify/slack/get    → GET  /api/v2/config/notify/slack
POST /uiapi/system/notify/slack/update → POST /api/v2/config/notify/slack
```

---

## 四、请求参数格式变化

### 4.1 查询参数 (POST Body → GET Query)

**旧格式:**
```typescript
// ClusterID 大写开头
{ ClusterID: "xxx", Namespace: "default" }
```

**新格式:**
```typescript
// cluster_id 下划线分隔
?cluster_id=xxx&namespace=default
```

### 4.2 操作参数 (POST Body)

**旧格式:**
```typescript
{
  ClusterID: "xxx",
  Namespace: "default",
  Name: "pod-1"
}
```

**新格式:**
```typescript
{
  cluster_id: "xxx",
  namespace: "default",
  name: "pod-1"
}
```

---

## 五、修改记录

### 2026-01-17

- [x] 创建任务管理文档
- [x] 修改 `request.ts` - 响应拦截器适配，添加 put/del 方法
- [x] 修改 `types/common.ts` - 添加 snake_case 类型定义
- [x] 修改 `auth.ts` - 用户认证 API 适配
- [x] 修改 `cluster.ts` - 集群信息 API 适配
- [x] 修改 `pod.ts` - Pod 查询和操作 API 适配
- [x] 修改 `node.ts` - Node 查询和操作 API 适配
- [x] 修改 `deployment.ts` - Deployment 查询和操作 API 适配
- [x] 修改 `service.ts` - Service 查询 API 适配
- [x] 修改 `ingress.ts` - Ingress 查询 API 适配
- [x] 修改 `namespace.ts` - Namespace 和 ConfigMap 查询 API 适配
- [x] 修改 `event.ts` - Event 查询 API 适配
- [x] 修改 `config.ts` - 通知配置 API 适配

---

## 六、注意事项

1. **响应拦截器**: 新 API 不使用 `code: 20000`，需要根据 HTTP 状态码判断
2. **参数命名**: 新 API 使用 snake_case (cluster_id)，旧 API 使用 PascalCase (ClusterID)
3. **HTTP 方法**: 查询类 API 从 POST 改为 GET
4. **路径结构**: 资源路径扁平化，如 `/api/v2/pods` 而非 `/uiapi/cluster/pod/list`
5. **兼容函数**: 所有旧接口保留了 `@deprecated` 兼容函数，逐步迁移后可移除

---

## 七、后续工作

### 7.1 页面组件适配（待完成）

需要更新以下页面组件以使用新的 API 函数：

- [ ] 登录页面
- [ ] 集群概览页面
- [ ] Pod 列表/详情页面
- [ ] Node 列表/详情页面
- [ ] Deployment 列表/详情页面
- [ ] Service/Ingress/Namespace 页面
- [ ] 事件日志页面
- [ ] 通知配置页面

### 7.2 测试检查点

- [ ] 登录功能正常
- [ ] 集群列表正常显示
- [ ] Pod 列表和详情正常
- [ ] Node 列表和详情正常
- [ ] Deployment 列表和详情正常
- [ ] 扩缩容操作正常
- [ ] Pod 日志获取正常
- [ ] 通知配置正常
