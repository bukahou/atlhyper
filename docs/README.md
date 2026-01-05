# AtlHyper 文档中心

> 本目录包含 AtlHyper 项目的所有开发文档。

---

## 文档索引

### 通用文档

| 文档 | 说明 | 适用范围 |
|------|------|----------|
| [ARCHITECTURE.md](./ARCHITECTURE.md) | 系统架构文档 | 全项目 |
| [DEVELOPMENT_GUIDE.md](./DEVELOPMENT_GUIDE.md) | 开发规范指南 | 全项目 |

### 后端文档

| 文档 | 说明 |
|------|------|
| [backend/BACKEND_TASKS.md](./backend/BACKEND_TASKS.md) | 后端开发任务和规范 |

主要模块：
- `atlhyper_master` - 主控中心
- `atlhyper_agent` - 集群代理
- `atlhyper_metrics` - 指标采集器

### 前端文档

| 文档 | 说明 |
|------|------|
| [frontend/CLAUDE.md](./frontend/CLAUDE.md) | AI 开发上下文（入口文件） |
| [frontend/ai-driven-development/](./frontend/ai-driven-development/) | AI 驱动开发规范 |
| [frontend/task-management/](./frontend/task-management/) | 任务管理 |

**前端文档详细索引：**

| 文档 | 路径 | 用途 |
|------|------|------|
| 开发指南 | `frontend/ai-driven-development/00-development-guide.md` | AI 开发规范 |
| 功能规格 | `frontend/ai-driven-development/01-spec.md` | 功能需求定义 |
| API 参考 | `frontend/ai-driven-development/02-api-reference.md` | 后端接口文档 |
| 开发进度 | `frontend/ai-driven-development/03-progress.md` | 进度追踪 |
| 任务列表 | `frontend/task-management/master-task-list.md` | 总任务管理 |

---

## 目录结构

```
docs/
├── README.md                 # 本文件（文档索引）
├── ARCHITECTURE.md           # 系统架构文档
├── DEVELOPMENT_GUIDE.md      # 开发规范指南
├── images/                   # 文档图片资源
│
├── backend/                  # 后端专用文档
│   └── BACKEND_TASKS.md      # 后端开发任务
│
└── frontend/                 # 前端专用文档
    ├── CLAUDE.md             # AI 开发上下文
    ├── ai-driven-development/    # AI 驱动开发规范
    │   ├── 00-development-guide.md
    │   ├── 01-spec.md
    │   ├── 02-api-reference.md
    │   └── 03-progress.md
    └── task-management/      # 任务管理
        └── master-task-list.md
```

---

## 文档分类说明

### 为什么前后端文档分开？

1. **技术栈差异**
   - 后端：Go + Gin + Kubernetes Client
   - 前端：Next.js + React + TypeScript

2. **开发规范差异**
   - 后端侧重：数据流、接口规范、并发安全
   - 前端侧重：组件设计、状态管理、UI/UX

3. **AI 开发规范差异**
   - 后端：需要理解 Kubernetes 概念、Go 并发模式
   - 前端：需要理解 React 模式、Next.js 路由

4. **任务管理独立**
   - 前后端可能并行开发
   - 各自追踪进度更清晰

### 统一的内容

- **架构文档**：全局视角，前后端都需要了解
- **开发规范**：核心设计理念、数据流规范
- **API 接口**：后端提供，前端消费

---

## 快速入口

### 新开发者

1. 阅读 [ARCHITECTURE.md](./ARCHITECTURE.md) 了解系统架构
2. 阅读 [DEVELOPMENT_GUIDE.md](./DEVELOPMENT_GUIDE.md) 了解开发规范
3. 根据工作方向选择前端或后端文档

### 前端开发

1. 阅读 [frontend/CLAUDE.md](./frontend/CLAUDE.md) 获取上下文
2. 查看 [frontend/ai-driven-development/03-progress.md](./frontend/ai-driven-development/03-progress.md) 了解当前进度
3. 参考 [frontend/ai-driven-development/02-api-reference.md](./frontend/ai-driven-development/02-api-reference.md) 了解 API

### 后端开发

1. 阅读 [DEVELOPMENT_GUIDE.md](./DEVELOPMENT_GUIDE.md) 了解开发规范
2. 重点关注：数据存储规范、接口调用规范、数据流规范
3. 参考 [ARCHITECTURE.md](./ARCHITECTURE.md) 了解模块详解

---

## 版本历史

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2025-01-04 | 初始版本，整合前后端文档 |
