# AtlHyper

Kubernetes 集群监控与管理平台

## 项目简介

AtlHyper 是一个现代化的 Kubernetes 集群监控与管理平台，提供直观的 Web 界面用于查看和管理多个 Kubernetes 集群的资源状态。

## 技术栈

### 后端
- **语言**: Go 1.21+
- **Web 框架**: Gin
- **数据库**: SQLite
- **认证**: JWT (HS256)

### 前端
- **框架**: Next.js 16 (App Router)
- **语言**: TypeScript
- **样式**: Tailwind CSS 4
- **状态管理**: Zustand
- **图表**: ECharts
- **图标**: Lucide React

### Agent
- **语言**: Go
- **通信**: gRPC

## 项目结构

```
atlhyper/
├── atlhyper_master/     # 主控端 (Master)
│   ├── server/          # HTTP API 服务
│   ├── db/              # 数据库层
│   ├── config/          # 配置管理
│   └── control/         # 集群控制逻辑
│
├── atlhyper_agent/      # 集群代理 (Agent)
│   ├── grpc/            # gRPC 服务
│   └── collector/       # 数据采集
│
├── atlhyper_metrics/    # 指标采集模块
│
├── atlhyper_web/        # Web 前端
│   ├── src/app/         # 页面路由
│   ├── src/components/  # 组件库
│   ├── src/api/         # API 调用
│   ├── src/store/       # 状态管理
│   ├── src/hooks/       # 自定义 Hooks
│   ├── src/i18n/        # 国际化 (中/日)
│   └── src/types/       # TypeScript 类型
│
├── model/               # 共享数据模型
├── common/              # 公共工具库
├── deploy/              # 部署配置
└── docs/                # 文档
```

## 功能模块

### 集群管理
- 多集群注册与管理
- Agent 状态监控
- 通知配置 (Slack)

### 资源监控
- **Pod**: 列表、详情、日志查看、重启
- **Node**: 列表、详情、隔离/解除隔离
- **Deployment**: 列表、详情、副本调整、镜像更新
- **Service**: 列表、详情
- **Namespace**: 列表、详情、ConfigMap 查看
- **Ingress**: 列表、详情

### 系统管理
- **指标概览**: 集群资源使用情况
- **日志管理**: 待开发
- **告警管理**: 待开发

### 用户与权限
- **用户管理**: 创建、角色修改、删除 (仅 Admin)
- **角色权限**: Viewer / Operator / Admin 三级权限
- **审计日志**: 操作记录与追溯

### 国际化
- 简体中文 (zh)
- 日本語 (ja)

## 快速开始

### 环境要求
- Go 1.21+
- Node.js 18+
- pnpm / npm

### 启动后端

```bash
cd atlhyper_master
go run main.go
```

### 启动前端

```bash
cd atlhyper_web
npm install
npm run dev
```

### 默认账户

| 用户名 | 密码 | 角色 |
|--------|------|------|
| admin  | admin123 | Admin |

## API 概览

### 公开接口
- `POST /uiapi/auth/login` - 用户登录
- `GET /uiapi/auth/user/list` - 用户列表
- `POST /uiapi/*/overview` - 资源概览
- `POST /uiapi/*/detail` - 资源详情

### 需登录接口
- `POST /uiapi/ops/pod/restart` - 重启 Pod
- `POST /uiapi/ops/pod/logs` - 查看日志
- `POST /uiapi/ops/node/cordon` - 隔离节点
- `POST /uiapi/ops/workload/scale` - 调整副本数

### 管理员接口
- `POST /uiapi/auth/user/register` - 创建用户
- `POST /uiapi/auth/user/update-role` - 修改角色
- `POST /uiapi/auth/user/delete` - 删除用户

## 配置

### 后端配置 (config.yaml)

```yaml
server:
  port: 8080

jwt:
  secret: "your-secret-key"
  expire_hours: 24
  min_password_len: 6
```

### 前端环境变量 (.env)

```
NEXT_PUBLIC_API_BASE_URL=http://localhost:8080
```

## 开发规范

### 后端
- 遵循 Go 标准项目布局
- 使用统一的响应格式 (`response.Success/Error`)
- Handler → Repository 分层架构
- 完整的注释文档

### 前端
- 组件化开发，按功能模块组织
- 使用 TypeScript 强类型
- i18n 支持多语言
- 响应式设计，支持移动端

## License

MIT
