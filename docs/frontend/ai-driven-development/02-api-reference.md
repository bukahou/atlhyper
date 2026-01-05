# AtlHyper Web - API 参考文档

## 概述

本文档记录 AtlHyper Master 后端提供的真实 API 接口，基于实际测试验证。

**基础 URL**: `http://localhost:8080` (开发环境)
**通用前缀**: `/uiapi`
**端口说明**:
- Master: 8080
- Agent: 8082

---

## 通用规范

### 认证方式

```
Authorization: Bearer {token}
```

> **重要**: 使用 `Authorization: Bearer` 而非 `X-Token`

### 响应格式

```typescript
interface ApiResponse<T = unknown> {
  code: number;       // 20000 = 成功
  message: string;    // 响应消息
  data: T;           // 响应数据
}
```

### 错误码

| 代码 | 含义 |
|------|------|
| 20000 | 成功 |
| 50008 | 非法 Token |
| 50012 | 其他客户端已登录 |
| 50014 | Token 过期 |

---

## 认证 API

### 用户登录

**POST** `/uiapi/auth/login`

> **注意**: 路径是 `/uiapi/auth/login`，不是 `/api/uiapi/user/login`

**请求体**:
```json
{
  "username": "admin",
  "password": "123456"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "登录成功",
  "data": {
    "cluster_ids": ["ZGMF-X10A"],
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "displayName": "Atlhyper",
      "id": 1,
      "role": 3,
      "username": "admin"
    }
  }
}
```

**TypeScript 类型**:
```typescript
interface LoginRequest {
  username: string;
  password: string;
}

interface LoginResponse {
  cluster_ids: string[];
  token: string;
  user: {
    displayName: string;
    id: number;
    role: number;  // 1=viewer, 2=operator, 3=admin
    username: string;
  };
}
```

---

### 获取用户列表

**GET** `/uiapi/auth/user/list`

**响应**:
```json
{
  "code": 20000,
  "message": "获取用户列表成功",
  "data": [
    {
      "ID": 1,
      "Username": "admin",
      "PasswordHash": "",
      "DisplayName": "Atlhyper",
      "Email": "admin@example.com",
      "Role": 3,
      "CreatedAt": "2025-09-13T17:32:00+09:00",
      "LastLogin": null
    }
  ]
}
```

---

### 获取审计日志

**GET** `/uiapi/auth/userauditlogs/list`

**响应**:
```json
{
  "code": 20000,
  "message": "获取用户审计日志成功",
  "data": [
    {
      "ID": 3,
      "UserID": 0,
      "Username": "anonymous",
      "Role": 1,
      "Action": "auto.uiapi/cluster/overview",
      "Success": false,
      "IP": "127.0.0.1",
      "Method": "POST",
      "Status": 401,
      "Timestamp": "2026-01-03T18:02:15+09:00"
    }
  ]
}
```

---

## 集群 API

### 集群概览

**POST** `/uiapi/cluster/overview`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "获取集群概览成功",
  "data": {
    "clusterId": "ZGMF-X10A",
    "cards": {
      "clusterHealth": {
        "podReadyPercent": 94,
        "nodeReadyPercent": 100,
        "status": "Healthy"
      },
      "nodeReady": {
        "total": 6,
        "ready": 6,
        "percent": 100
      },
      "cpuUsage": {
        "percent": 4.41
      },
      "memUsage": {
        "percent": 15.07
      },
      "events24h": 3
    },
    "trends": {
      "resourceUsage": [
        {
          "at": "2026-01-03T09:07:00Z",
          "cpuPeak": 0,
          "cpuPeakNode": "",
          "memPeak": 0,
          "memPeakNode": "",
          "tempPeak": 0,
          "tempPeakNode": ""
        }
      ]
    },
    "alerts": {
      "totals": {
        "critical": 0,
        "warning": 0,
        "info": 3
      },
      "trend": [],
      "recent": []
    },
    "nodes": {
      "usage": [
        {
          "node": "desk-one",
          "cpuUsage": 3.6,
          "memUsage": 15.43
        }
      ]
    }
  }
}
```

---

## Pod API

### Pod 概览

**POST** `/uiapi/pod/overview`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "获取 Pod 概览成功",
  "data": {
    "cards": {
      "running": 47,
      "pending": 0,
      "failed": 0,
      "unknown": 3
    },
    "pods": [
      {
        "namespace": "atlhyper",
        "deployment": "atlhyper-agent-798f994bc",
        "name": "atlhyper-agent-798f994bc-mbkq9",
        "ready": "1/1",
        "phase": "Running",
        "restarts": 3,
        "cpu": 12,
        "cpuPercent": 12.7,
        "memory": 30,
        "memPercent": 23.3,
        "cpuText": "12000m",
        "cpuPercentText": "12.700%",
        "memoryText": "30 m",
        "memPercentText": "23.300%",
        "startTime": "2025-10-04T20:31:02+09:00",
        "node": "raspi-one"
      }
    ]
  }
}
```

**TypeScript 类型**:
```typescript
interface PodOverviewResponse {
  cards: {
    running: number;
    pending: number;
    failed: number;
    unknown: number;
  };
  pods: PodItem[];
}

interface PodItem {
  namespace: string;
  deployment: string;
  name: string;
  ready: string;
  phase: string;
  restarts: number;
  cpu: number;
  cpuPercent: number;
  memory: number;
  memPercent: number;
  cpuText: string;
  cpuPercentText: string;
  memoryText: string;
  memPercentText: string;
  startTime: string;
  node: string;
}
```

---

### Pod 详情

**POST** `/uiapi/pod/detail`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Namespace": "default",
  "PodName": "nginx-xxx"
}
```

---

### Pod 日志

**POST** `/uiapi/ops/pod/logs`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Namespace": "default",
  "Pod": "nginx-xxx",
  "TailLines": 100
}
```

---

### 重启 Pod

**POST** `/uiapi/ops/pod/restart`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Namespace": "default",
  "Pod": "nginx-xxx"
}
```

---

## Node API

### Node 概览

**POST** `/uiapi/node/overview`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "node overview retrieved successfully",
  "data": {
    "cards": {
      "totalNodes": 6,
      "readyNodes": 6,
      "totalCPU": 36,
      "totalMemoryGiB": 116.9
    },
    "rows": [
      {
        "name": "desk-one",
        "ready": true,
        "internalIP": "2408:210:ba06:d800:12e7:c6ff:fe08:d4bf",
        "osImage": "Ubuntu 24.04.3 LTS",
        "architecture": "amd64",
        "cpuCores": 8,
        "memoryGiB": 31.2,
        "schedulable": true
      }
    ]
  }
}
```

**TypeScript 类型**:
```typescript
interface NodeOverviewResponse {
  cards: {
    totalNodes: number;
    readyNodes: number;
    totalCPU: number;
    totalMemoryGiB: number;
  };
  rows: NodeItem[];
}

interface NodeItem {
  name: string;
  ready: boolean;
  internalIP: string;
  osImage: string;
  architecture: string;
  cpuCores: number;
  memoryGiB: number;
  schedulable: boolean;
}
```

---

### Node 详情

**POST** `/uiapi/node/detail`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "NodeName": "node-1"
}
```

---

### Node Cordon

**POST** `/uiapi/ops/node/cordon`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Node": "node-1"
}
```

> **注意**: 参数名是 `Node` 而不是 `NodeName`

---

### Node Uncordon

**POST** `/uiapi/ops/node/uncordon`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Node": "node-1"
}
```

---

## Deployment API

### Deployment 概览

**POST** `/uiapi/deployment/overview`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "获取 Deployment 概览成功",
  "data": {
    "cards": {
      "totalDeployments": 23,
      "namespaces": 9,
      "totalReplicas": 23,
      "readyReplicas": 23
    },
    "rows": [
      {
        "namespace": "atlhyper",
        "name": "atlhyper-agent",
        "image": "bukahou/atlhyper-agent:v1.1.0",
        "replicas": "1/1",
        "labelCount": 1,
        "annoCount": 2,
        "createdAt": "2025-10-04T20:31:02+09:00"
      }
    ]
  }
}
```

---

### Deployment 详情

**POST** `/uiapi/deployment/detail`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Namespace": "default",
  "Name": "nginx"
}
```

---

### 更新镜像

**POST** `/uiapi/ops/workload/updateImage`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Namespace": "default",
  "Kind": "Deployment",
  "Name": "nginx",
  "NewImage": "nginx:1.21",
  "OldImage": "nginx:1.20"
}
```

---

### 扩缩容

**POST** `/uiapi/ops/workload/scale`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Namespace": "default",
  "Kind": "Deployment",
  "Name": "nginx",
  "Replicas": 3
}
```

---

## Service API

### Service 概览

**POST** `/uiapi/service/overview`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "获取 Service 概览成功",
  "data": {
    "cards": {
      "totalServices": 24,
      "externalServices": 2,
      "internalServices": 22,
      "headlessServices": 0
    },
    "rows": [
      {
        "name": "atlhyper-controller",
        "namespace": "atlhyper",
        "type": "ClusterIP",
        "clusterIP": "10.43.28.47",
        "ports": "8081:8081",
        "protocol": "TCP",
        "selector": "app=atlhyper-controller",
        "createdAt": "2025-10-04T20:30:27+09:00"
      }
    ]
  }
}
```

---

## Namespace API

### Namespace 概览

**POST** `/uiapi/namespace/overview`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "命名空间概览获取成功",
  "data": {
    "cards": {
      "totalNamespaces": 14,
      "activeCount": 14,
      "terminating": 0,
      "totalPods": 50
    },
    "rows": [
      {
        "name": "atlhyper",
        "status": "Active",
        "podCount": 8,
        "labelCount": 1,
        "annotationCount": 1,
        "createdAt": "2025-10-04T20:30:27+09:00"
      }
    ]
  }
}
```

---

### Namespace 详情

**POST** `/uiapi/namespace/detail`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Namespace": "default"
}
```

---

### ConfigMap 详情

**POST** `/uiapi/configmap/detail`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "Namespace": "default"
}
```

---

## Ingress API

### Ingress 概览

**POST** `/uiapi/ingress/overview`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "获取 Ingress 概览成功",
  "data": {
    "cards": {
      "totalIngresses": 6,
      "usedHosts": 7,
      "tlsCerts": 6,
      "totalPaths": 9
    },
    "rows": [
      {
        "name": "atlhyper-controller-ingress",
        "namespace": "atlhyper",
        "host": "atlhyper.com",
        "path": "/",
        "serviceName": "atlhyper-controller",
        "servicePort": "8081",
        "tls": "atlhyper.com, www.atlhyper.com",
        "createdAt": "2025-10-20T09:44:01+09:00"
      }
    ]
  }
}
```

---

## 事件日志 API

### 事件列表

**POST** `/uiapi/event/logs`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "WithinDays": 7
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "获取事件日志成功",
  "data": {
    "cards": {
      "totalAlerts": 22,
      "totalEvents": 4,
      "warning": 3,
      "info": 12,
      "error": 0,
      "categoriesCount": 5,
      "kindsCount": 3
    },
    "rows": [
      {
        "ClusterID": "ZGMF-X10A",
        "Category": "Terminated",
        "EventTime": "2026-01-03T18:07:16+09:00",
        "Kind": "Pod",
        "Message": "容器已正常退出（用于 Job）",
        "Name": "helm-install-traefik-nqw6p",
        "Namespace": "kube-system",
        "Node": "desk-zero",
        "Reason": "Completed",
        "Severity": "info",
        "Time": "2026-01-03T18:07:24+09:00"
      }
    ]
  }
}
```

---

## 配置 API

### 获取 Slack 配置

**POST** `/uiapi/config/slack/get`

**响应**:
```json
{
  "code": 20000,
  "message": "OK",
  "data": {
    "ID": 1,
    "Name": "slack",
    "Enable": 0,
    "Webhook": "https://hooks.slack.com/services/...",
    "IntervalSec": 5,
    "UpdatedAt": "2025-09-13T17:32:00+09:00"
  }
}
```

---

### 更新 Slack 配置

**POST** `/uiapi/config/slack/update`

**请求体**:
```json
{
  "enable": 1,
  "webhook": "https://hooks.slack.com/...",
  "intervalSec": 60
}
```

---

## 指标 API

### 指标概览

**POST** `/uiapi/metrics/overview`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A"
}
```

**响应**:
```json
{
  "code": 20000,
  "message": "获取集群指标概览成功",
  "data": {
    "cards": {
      "avgCPUPercent": 0,
      "avgMemPercent": 0,
      "peakTempC": 0,
      "peakTempNode": "",
      "peakDiskPercent": 0,
      "peakDiskNode": ""
    },
    "rows": []
  }
}
```

---

### 节点指标详情

**POST** `/uiapi/metrics/node/detail`

**请求体**:
```json
{
  "ClusterID": "ZGMF-X10A",
  "NodeID": "desk-one"
}
```

---

## 待办事项 API

### 获取所有待办事项

**GET** `/uiapi/user/todos/all`

**响应**:
```json
{
  "code": 20000,
  "message": "获取成功",
  "data": {
    "items": [
      {
        "id": 1,
        "username": "admin",
        "title": "欢迎使用AtlHyper",
        "content": "这是系统自动生成的第一条代办事项",
        "created_at": "2025-09-13 17:32:00",
        "updated_at": "2025-09-13 08:32:00",
        "is_done": 0,
        "due_date": null,
        "priority": 1,
        "category": "系统初始化",
        "deleted": 0
      }
    ],
    "total": 1
  }
}
```

---

### 创建待办事项

**POST** `/uiapi/user/todo/create`

**请求体**:
```json
{
  "username": "admin",
  "title": "任务标题",
  "content": "任务内容",
  "is_done": 0,
  "priority": 2,
  "category": "工作",
  "due_date": "2026-01-10"
}
```

---

### 更新待办事项

**POST** `/uiapi/user/todo/update`

**请求体**:
```json
{
  "id": 1,
  "title": "新标题",
  "is_done": 1
}
```

---

### 删除待办事项

**POST** `/uiapi/user/todo/delete`

**请求体**:
```json
{
  "id": 1
}
```

---

## 前端 API 问题汇总

### 已发现的差异

| 问题 | 当前实现 | 正确实现 |
|------|---------|---------|
| 认证头 | `X-Token` | `Authorization: Bearer` |
| 登录路径 | `/api/uiapi/user/login` | `/uiapi/auth/login` |
| 登录响应类型 | 只有 token | 包含 cluster_ids, token, user |
| UserInfo 类型 | name, avatar, roles[] | displayName, username, id, role (number) |

### 需要修复的文件

1. `src/api/request.ts` - 修改认证头
2. `src/api/auth.ts` - 修改登录路径
3. `src/types/auth.ts` - 修改类型定义
4. `src/store/authStore.ts` - 适配新的响应格式

---

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2026-01-03 | 初始版本 |
| 2.0 | 2026-01-03 | 基于实际 API 测试重写，修正所有接口定义 |
