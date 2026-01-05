# AtlHyper 部署配置

本目录包含 AtlHyper 项目的所有部署相关文件。

## 目录结构

```
deploy/
├── docker/                     # Dockerfile 文件
│   ├── Dockerfile.controller   # 后端控制器 (API)
│   ├── Dockerfile.agent        # Agent 数据采集代理
│   ├── Dockerfile.metrics      # 指标采集服务
│   └── Dockerfile.web          # 前端 Web 应用 (Next.js)
├── k8s/                        # Kubernetes 部署配置
│   ├── atlhyper-controller.yaml
│   ├── atlhyper-agent.yaml
│   ├── atlhyper-metrics.yaml
│   └── atlhyper-web.yaml
├── scripts/                    # 构建脚本
│   ├── _common.sh              # 公共构建函数
│   ├── build_controller.sh
│   ├── build_agent.sh
│   ├── build_metrics.sh
│   └── build_web.sh
└── README.md
```

## 构建镜像

### 修改版本标签

每个构建脚本顶部定义了 `TAG` 变量，修改即可切换版本：

```bash
# 在脚本顶部修改
TAG="v1.0.0"   # 正式发布版本
# TAG="latest" # 最新稳定版
# TAG="test"   # 测试环境
```

### 执行构建

```bash
# 构建 Controller (后端 API)
./deploy/scripts/build_controller.sh

# 构建 Web Frontend (Next.js)
./deploy/scripts/build_web.sh

# 构建 Agent
./deploy/scripts/build_agent.sh

# 构建 Metrics
./deploy/scripts/build_metrics.sh
```

所有脚本使用 Docker Buildx 构建多架构镜像 (`linux/amd64`, `linux/arm64`) 并自动推送到 Docker Hub。

## 部署到 Kubernetes

### 1. 创建命名空间和配置

```bash
# 部署 Controller (会自动创建 namespace)
kubectl apply -f deploy/k8s/atlhyper-controller.yaml
```

> **重要**: 部署前请修改 `atlhyper-controller-secret` 中的敏感信息！

### 2. 部署其他服务

```bash
# 部署 Web Frontend
kubectl apply -f deploy/k8s/atlhyper-web.yaml

# 部署 Agent (集群数据采集)
kubectl apply -f deploy/k8s/atlhyper-agent.yaml

# 部署 Metrics (节点指标采集)
kubectl apply -f deploy/k8s/atlhyper-metrics.yaml
```

### 3. 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n atlhyper

# 查看服务
kubectl get svc -n atlhyper

# 查看日志
kubectl logs -f deployment/atlhyper-controller -n atlhyper
```

## 服务端口

| 服务 | 端口 | 描述 |
|------|------|------|
| Controller | 8080 | 后端 API |
| Web | 3000 | 前端 UI |
| Agent | 8082 | 数据采集 |
| Metrics | 8083 | 指标采集 |

## 架构说明

```
                    ┌─────────────┐
                    │   Ingress   │
                    └──────┬──────┘
                           │
               ┌───────────┴───────────┐
               │                       │
               ▼                       ▼
         ┌──────────┐           ┌──────────┐
         │   Web    │           │Controller│
         │ (Next.js)│──────────▶│ (Go API) │
         └──────────┘           └────┬─────┘
                                     │
                                     ▼
                              ┌──────────┐
                              │  Agent   │
                              └────┬─────┘
                                   │
              ┌────────────────────┼────────────────────┐
              │                    │                    │
              ▼                    ▼                    ▼
        ┌──────────┐         ┌──────────┐         ┌──────────┐
        │ Metrics  │         │ Metrics  │         │ Metrics  │
        │ (Node 1) │         │ (Node 2) │         │ (Node N) │
        └──────────┘         └──────────┘         └──────────┘
```

## 环境变量

### Controller

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `DEFAULT_ADMIN_USERNAME` | 管理员用户名 | admin |
| `DEFAULT_ADMIN_PASSWORD` | 管理员密码 | admin123 |
| `JWT_SECRET` | JWT 签名密钥 | - |
| `JWT_EXPIRE_HOURS` | Token 过期时间(小时) | 24 |
| `ENABLE_EMAIL_ALERT` | 启用邮件告警 | false |
| `ENABLE_SLACK_ALERT` | 启用 Slack 告警 | false |

### Web

| 变量 | 描述 | 默认值 |
|------|------|--------|
| `NEXT_PUBLIC_API_URL` | 后端 API 地址 | - |

## 镜像信息

- **Registry**: Docker Hub
- **Namespace**: bukahou
- **多架构支持**: `linux/amd64`, `linux/arm64`

镜像列表：
- `bukahou/atlhyper-controller`
- `bukahou/atlhyper-web`
- `bukahou/atlhyper-agent`
- `bukahou/atlhyper-metrics`

## 注意事项

1. **前后端解耦**: Controller 不再包含前端构建，Web 独立部署
2. **敏感信息**: 生产环境请使用 Kubernetes Secret 管理敏感配置
3. **健康检查**: 所有服务都配置了 `/health` 端点用于健康检查
4. **数据持久化**: Controller 需要持久化存储用于 SQLite 数据库
