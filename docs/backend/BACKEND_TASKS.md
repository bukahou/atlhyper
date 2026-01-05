# AtlHyper 后端开发任务

> 本文档记录后端优化的任务列表和开发规范。

---

## 当前任务

### TASK-001: Config 配置文件整理

**状态**: ✅ 已完成
**优先级**: P0
**影响范围**: 全模块
**完成日期**: 2025-01-04

#### 问题分析

当前 config 文件分布混乱，存在跨模块依赖问题：

**现状：**
```
AtlHyper/
├── config/config.go              # root 级别配置（被标记为 Master 配置）
│   ├── DiagnosisConfig           # ⚠️ 被 Agent 错误引用
│   ├── KubernetesConfig
│   ├── MailerConfig
│   ├── SlackConfig
│   ├── WebhookConfig
│   ├── ServerConfig
│   └── AdminConfig
│
├── atlhyper_metrics/config/      # Metrics 模块自有配置
│   └── config.go
│       └── PushConfig            # 推送到 Agent 的配置
│
└── atlhyper_agent/external/push/config/  # Agent 推送配置（分散）
    ├── restclient_config.go      # REST 客户端配置
    └── routes.go                 # 路由常量
```

**问题点：**

| 问题 | 文件 | 说明 |
|------|------|------|
| 跨模块依赖 | `atlhyper_agent/internal/diagnosis/cleaner.go` | Agent 引用了 `AtlHyper/config`（Master 的配置） |
| 配置分散 | Agent 配置散落在多处 | 不便于统一管理 |
| 职责不清 | root/config 定位模糊 | 既有 Master 配置，又被 Agent 引用 |

#### 方案对比

| 方案 | 优点 | 缺点 | 适用场景 |
|------|------|------|----------|
| **A: 统一管理** | 配置集中，便于查找修改 | 模块耦合，无法独立部署 | 单体应用 |
| **B: 模块自治** | 解耦，可独立部署配置 | 配置分散，可能重复 | 分布式部署 |

#### 推荐方案

**推荐方案 B：模块自治**

理由：
1. 四层架构（WebUI → Master → Agent → Metrics）要求模块可独立部署
2. Agent 和 Master 部署环境不同（集群内 vs 集群外）
3. 各模块配置差异大，强行统一会增加复杂度

#### 目标结构

```
AtlHyper/
├── atlhyper_master/config/       # Master 专用配置
│   ├── config.go                 # 主配置结构
│   ├── diagnosis.go              # 诊断相关配置
│   ├── mailer.go                 # 邮件配置
│   ├── slack.go                  # Slack 配置
│   └── server.go                 # 服务器配置
│
├── atlhyper_agent/config/        # Agent 专用配置
│   ├── config.go                 # 主配置结构
│   ├── push.go                   # 推送配置
│   └── diagnosis.go              # Agent 侧诊断配置
│
├── atlhyper_metrics/config/      # Metrics 专用配置（已存在）
│   └── config.go
│
└── config/                       # 可删除或保留为共享常量
```

#### 迁移步骤

1. **创建各模块 config 目录**
2. **迁移配置到对应模块**
3. **修复 Agent 对 root/config 的引用**
4. **更新所有 import 路径**
5. **删除 root/config（或保留为共享常量）**
6. **验证构建和测试**

#### 决策待定

- [ ] 确认采用方案 B（模块自治）
- [ ] 确定 root/config 是否保留（可作为共享常量目录）
- [ ] 确定配置文件格式（纯 Go struct vs YAML/JSON）

---

## 任务列表

| ID | 任务 | 优先级 | 状态 | 依赖 |
|----|------|--------|------|------|
| TASK-001 | Config 配置文件整理 | P0 | ✅ 已完成 | - |
| TASK-002 | 修复 Agent 引用 Master 配置问题 | P0 | ✅ 已完成（随 TASK-001 完成） | TASK-001 |
| TASK-003 | 统一环境变量命名规范 | P1 | ✅ 已完成（随 TASK-001 完成） | TASK-001 |
| TASK-004 | 添加配置校验机制 | P2 | 待定 | TASK-001 |

---

## 开发规范

### 配置管理规范

#### 1. 模块自治原则

每个模块维护自己的配置，不跨模块引用配置。

```go
// ✅ 正确：Agent 引用自己的 config
import "AtlHyper/atlhyper_agent/config"

// ❌ 错误：Agent 引用 Master 的 config
import "AtlHyper/config"
```

#### 2. 环境变量命名规范

```
{MODULE}_{CATEGORY}_{NAME}

示例：
MASTER_SERVER_PORT=8080
MASTER_MAILER_SMTP_HOST=smtp.gmail.com
AGENT_PUSH_INTERVAL=5s
AGENT_API_BASE_URL=http://master:8080
METRICS_PUSH_URL=http://agent:8082
```

#### 3. 必须设置默认值

```go
var defaultStrings = map[string]string{
    "MASTER_SERVER_PORT": "8080",  // 必须有默认值
}
```

#### 4. 配置加载模式

```go
// 推荐模式
func LoadConfig() error {
    // 1. 读取环境变量
    // 2. 应用默认值
    // 3. 校验配置
    // 4. 返回错误或赋值到全局变量
}

func MustLoad() {
    if err := LoadConfig(); err != nil {
        panic(err)
    }
}
```

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2025-01-04 | 初始版本，添加 Config 整理任务 |
