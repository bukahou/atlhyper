# AtlHyper Master API 参考文档

## 认证策略

AtlHyper 采用分级权限策略：

| 级别 | Role 值 | 说明 |
|------|---------|------|
| **Public** | - | 无需登录，所有只读查询对外开放 |
| **Viewer** | 1 | 等同游客，可使用 AI 对话 |
| **Operator** | 2 | 敏感信息查看、指令下发 |
| **Admin** | 3 | 用户管理、系统配置 |

## API 端点列表

### 1. 健康检查

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/health` | Public | ❌ | 健康检查 |

---

### 2. 用户认证

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| POST | `/api/v2/user/login` | Public | ✅ | 用户登录 |

---

### 3. 集群管理

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/overview` | Public | ❌ | 获取集群概览（Dashboard 数据） |
| GET | `/api/v2/clusters` | Public | ❌ | 获取集群列表 |
| GET | `/api/v2/clusters/{id}` | Public | ❌ | 获取指定集群详情 |

---

### 4. 工作负载查询

#### 4.1 Pod

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/pods` | Public | ❌ | 获取 Pod 列表 |
| GET | `/api/v2/pods/{namespace}/{name}` | Public | ❌ | 获取 Pod 详情 |

#### 4.2 Node

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/nodes` | Public | ❌ | 获取节点列表 |
| GET | `/api/v2/nodes/{name}` | Public | ❌ | 获取节点详情 |

#### 4.3 Deployment

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/deployments` | Public | ❌ | 获取 Deployment 列表 |
| GET | `/api/v2/deployments/{namespace}/{name}` | Public | ❌ | 获取 Deployment 详情 |

#### 4.4 DaemonSet

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/daemonsets` | Public | ❌ | 获取 DaemonSet 列表 |
| GET | `/api/v2/daemonsets/{namespace}/{name}` | Public | ❌ | 获取 DaemonSet 详情 |

#### 4.5 StatefulSet

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/statefulsets` | Public | ❌ | 获取 StatefulSet 列表 |
| GET | `/api/v2/statefulsets/{namespace}/{name}` | Public | ❌ | 获取 StatefulSet 详情 |

---

### 5. 网络资源查询

#### 5.1 Service

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/services` | Public | ❌ | 获取 Service 列表 |
| GET | `/api/v2/services/{namespace}/{name}` | Public | ❌ | 获取 Service 详情 |

#### 5.2 Ingress

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/ingresses` | Public | ❌ | 获取 Ingress 列表 |
| GET | `/api/v2/ingresses/{namespace}/{name}` | Public | ❌ | 获取 Ingress 详情 |

---

### 6. 配置资源

#### 6.1 ConfigMap

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/configmaps` | Public | ❌ | 获取 ConfigMap 列表 |
| GET | `/api/v2/configmaps/{namespace}/{name}` | Operator | ❌ | 获取 ConfigMap 详情 |
| GET | `/api/v2/ops/configmaps/data` | Operator | ✅ | 获取 ConfigMap 数据（敏感） |

#### 6.2 Secret

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/secrets` | Operator | ❌ | 获取 Secret 列表 |
| GET | `/api/v2/ops/secrets/data` | Operator | ✅ | 获取 Secret 数据（敏感） |

---

### 7. Namespace

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/namespaces` | Public | ❌ | 获取命名空间列表 |
| GET | `/api/v2/namespaces/{name}` | Public | ❌ | 获取命名空间详情 |

---

### 8. 事件

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/events` | Public | ❌ | 获取事件列表 |
| GET | `/api/v2/events/by-resource` | Public | ❌ | 按资源获取事件 |

---

### 9. 指令系统

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/commands/history` | Public | ❌ | 获取指令历史 |
| GET | `/api/v2/commands/{id}` | Public | ❌ | 获取指令状态 |
| POST | `/api/v2/commands` | Operator | ✅ | 创建/下发指令 |

---

### 10. 操作接口 (Ops)

#### 10.1 Pod 操作

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| POST | `/api/v2/ops/pods/logs` | Operator | ✅ | 获取 Pod 日志 |
| POST | `/api/v2/ops/pods/restart` | Operator | ✅ | 重启 Pod |

#### 10.2 Deployment 操作

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| POST | `/api/v2/ops/deployments/scale` | Operator | ✅ | 扩缩容 Deployment |
| POST | `/api/v2/ops/deployments/restart` | Operator | ✅ | 重启 Deployment |
| POST | `/api/v2/ops/deployments/image` | Operator | ✅ | 更新镜像 |

#### 10.3 Node 操作

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| POST | `/api/v2/ops/nodes/cordon` | Operator | ✅ | 标记节点不可调度 |
| POST | `/api/v2/ops/nodes/uncordon` | Operator | ✅ | 取消节点不可调度 |

---

### 11. 通知渠道

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/notify/channels` | Operator | ❌ | 获取通知渠道列表 |
| PUT | `/api/v2/notify/channels/{type}` | Operator | ✅ | 更新通知渠道 |
| GET | `/api/v2/notify/channels/{type}` | Operator | ❌ | 获取渠道详情 |
| POST | `/api/v2/notify/channels/{type}/test` | Operator | ❌ | 测试通知发送 |

---

### 12. 审计日志

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/audit/logs` | Operator | ❌ | 获取审计日志 |

---

### 13. SLO 监控

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/slo/domains` | Public | ❌ | 按 service key 获取域名列表（V1） |
| GET | `/api/v2/slo/domains/v2` | Public | ❌ | 按真实域名获取列表（V2） |
| GET | `/api/v2/slo/domains/detail` | Public | ❌ | 获取域名详情 |
| GET | `/api/v2/slo/domains/history` | Public | ❌ | 获取域名历史数据 |
| GET | `/api/v2/slo/targets` | Public | ❌ | 获取 SLO 目标配置 |
| GET | `/api/v2/slo/status-history` | Public | ❌ | 获取状态历史 |

---

### 14. 节点指标

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/node-metrics` | Public | ❌ | 获取集群所有节点指标 |
| GET | `/api/v2/node-metrics/{nodeName}` | Public | ❌ | 获取指定节点指标 |
| GET | `/api/v2/node-metrics/{nodeName}/history` | Public | ❌ | 获取节点历史指标 |

---

### 15. AI 配置

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/settings/ai` | Operator | ❌ | 获取 AI 配置（只读） |
| PUT | `/api/v2/settings/ai/{key}` | Admin | ✅ | 更新 AI 配置 |

---

### 16. AI Provider 管理

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/ai/providers` | Operator | ❌ | 获取 Provider 列表 |
| GET | `/api/v2/ai/active` | Operator | ❌ | 获取当前激活配置 |
| POST | `/api/v2/ai/providers` | Admin | ✅ | 创建 Provider |
| PUT | `/api/v2/ai/providers/{id}` | Admin | ✅ | 更新 Provider |
| DELETE | `/api/v2/ai/providers/{id}` | Admin | ✅ | 删除 Provider |
| PUT | `/api/v2/ai/active/{provider}` | Admin | ✅ | 设置激活 Provider |

---

### 17. AI 对话

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/ai/conversations` | Viewer+ | ❌ | 获取对话列表 |
| POST | `/api/v2/ai/conversations` | Viewer+ | ❌ | 创建新对话 |
| GET | `/api/v2/ai/conversations/{id}` | Viewer+ | ❌ | 获取对话详情 |
| DELETE | `/api/v2/ai/conversations/{id}` | Operator | ❌ | 删除对话 |
| GET | `/api/v2/ai/conversations/{id}/messages` | Viewer+ | ❌ | 获取对话消息 |
| POST | `/api/v2/ai/chat` | Viewer+ | ❌ | 发送消息（SSE 流式响应） |

---

### 18. 用户管理

| 方法 | 路径 | 权限 | 审计 | 说明 |
|------|------|------|------|------|
| GET | `/api/v2/user/list` | Admin | ❌ | 获取用户列表 |
| POST | `/api/v2/user/register` | Admin | ✅ | 注册新用户 |
| PUT | `/api/v2/user/update-role` | Admin | ✅ | 更新用户角色 |
| PUT | `/api/v2/user/update-status` | Admin | ✅ | 更新用户状态 |
| DELETE | `/api/v2/user/delete` | Admin | ✅ | 删除用户 |

---

## 通用查询参数

大多数列表 API 支持以下查询参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `cluster_id` | string | 集群 ID（必需） |
| `namespace` | string | 命名空间过滤 |
| `limit` | int | 返回数量限制 |
| `offset` | int | 分页偏移量 |

---

## 响应格式

### 成功响应

```json
{
  "data": { ... },
  "message": "success"
}
```

### 错误响应

```json
{
  "error": "error message",
  "code": 400
}
```

---

## 审计日志

所有标记为"审计 ✅"的接口会记录以下信息：

- 操作时间
- 操作用户
- 操作类型（create/read/update/delete/execute）
- 目标资源
- 请求详情
- 响应状态
- 客户端 IP

即使因权限不足而失败的请求也会被记录。
