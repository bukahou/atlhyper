# AtlHyper Web - AI 开发上下文

此文件为 AI 助手提供项目上下文，方便后续 Chat 继续开发。

## 项目概述

AtlHyper Web 是 AtlHyper Kubernetes 监控平台的新一代前端，采用 Next.js 16 + React 19 + TypeScript + Tailwind CSS v4 重构，实现与后端完全解耦。

**项目路径**: `/home/wuxiafeng/AtlHyper/GitHub/atlhyper/atlhyper_web/`

---

## 硬性要求

### 绝对禁止

1. **禁止修改现有代码** - 不能修改 `atlhyper/web/` 目录下的任何文件
2. **禁止修改后端代码** - 不能修改 atlhyper_master, atlhyper_agent 等
3. **只能复制参考** - 现有代码只能作为参考，不能直接编辑

### 原因

开发失败后无法回滚，会导致项目无法继续。

---

## 核心设计原则

### 访问控制策略

| 操作类型 | 需要登录 | 说明 |
|---------|---------|------|
| 浏览/查看 | 否 | 所有页面可直接访问，方便开源项目了解 |
| 修改/操作 | 是 | Pod 重启、Node drain 等操作需登录 |
| 配置变更 | 是 | Slack 配置、用户管理等需登录 |

**实现方式**: 操作按钮触发时检查登录状态，未登录时弹出登录对话框。

### 国际化

- 支持中文 (zh) 和日文 (ja)
- 默认语言: 中文
- 语言偏好存储在 localStorage

### 主题系统

- 支持亮色、暗色、跟随系统
- 主题色: **淡绿蓝色 (teal/cyan)**
- 主题偏好存储在 localStorage

### 品牌标识

| 项目 | 图标 | 主题色 |
|------|------|--------|
| Geass | geass 图标 | 淡紫色 |
| Atlantis | geass 图标 | 淡蓝色 |
| **AtlHyper** | **geass 图标** | **淡绿蓝色 (teal/cyan)** |

---

## 技术栈

| 技术 | 版本 |
|------|------|
| Next.js | 16.x |
| React | 19.x |
| TypeScript | 5.x |
| Tailwind CSS | 4.x |
| Zustand | 5.x |
| Lucide React | latest |
| ECharts | 5.x |
| axios | latest |

---

## 当前开发状态

**阶段**: 基础功能已完成，模块化优化中
**进度**: 48/48 任务完成，正在进行代码规范优化
**任务总数**: 48 个

### 已完成

- [x] Phase 0: 项目初始化
- [x] Phase 1: P0 核心功能 (I18n/Theme/Layout/Auth)
- [x] Phase 2: P1 核心监控 (Analysis/Pod/Node)
- [x] Phase 3: P2 扩展监控 (Deployment/Service/Namespace/Ingress/Alert)
- [x] Phase 4: P3 系统管理 (Metrics/Logs/Users/Audit)

### 当前任务

- [ ] 创建通用组件 (PageHeader/StatsCard/DataTable/StatusBadge)
- [ ] 重构页面使用通用组件
- [ ] 确保所有文件符合行数规范

---

## 功能需求概览 (16 个 FR)

| 优先级 | FR | 功能 | 需登录 |
|--------|-----|------|--------|
| P0 | FR-001 | 认证系统（操作级别） | 操作时 |
| P0 | FR-002 | 布局框架 | 否 |
| P0 | FR-014 | 国际化支持 (中日双语) | 否 |
| P0 | FR-015 | 主题切换 | 否 |
| P0 | FR-016 | 响应式布局 | 否 |
| P1 | FR-003 | Analysis 页面 | 否 |
| P1 | FR-004 | Workbench 页面 | 修改时 |
| P1 | FR-005 | Pod 监控 | 操作时 |
| P1 | FR-006 | Node 监控 | 操作时 |
| P2 | FR-007 | Deployment 监控 | 操作时 |
| P2 | FR-008 | Service 监控 | 否 |
| P2 | FR-009 | Namespace 监控 | 否 |
| P2 | FR-010 | Ingress 监控 | 否 |
| P2 | FR-011 | 告警监控 | 否 |
| P3 | FR-012 | 系统指标监控 | 否 |
| P3 | FR-013 | 用户管理 | 是 |

---

## 关键文档

| 文档 | 路径 | 用途 |
|------|------|------|
| 开发指南 | docs/ai-driven-development/00-development-guide.md | AI 开发规范 |
| 功能规格 | docs/ai-driven-development/01-spec.md | 功能需求定义 |
| API 参考 | docs/ai-driven-development/02-api-reference.md | 后端接口文档 |
| 开发进度 | docs/ai-driven-development/03-progress.md | 进度追踪 |
| 任务列表 | docs/task-management/master-task-list.md | 总任务管理 |

---

## 参考项目

| 项目 | 路径 | 参考内容 |
|------|------|---------|
| Geass | /home/wuxiafeng/AtlHyper/GitHub/Geass | AI 开发规范 |
| atlantis | /home/wuxiafeng/AtlHyper/GitHub/atlantis | 代码风格、i18n、主题 |
| atlhyper/web | /home/wuxiafeng/AtlHyper/GitHub/atlhyper/web | API 接口和业务逻辑 |

---

## 后端 API

- **基础 URL**: `http://localhost:8080`
- **通用前缀**: `/uiapi`
- **认证**: X-Token Header
- **成功码**: `code: 20000`

主要接口:
- 登录: POST `/api/uiapi/user/login`
- Pod 列表: POST `/uiapi/pod/overview`
- Node 列表: POST `/uiapi/node/overview`
- 详见: `docs/ai-driven-development/02-api-reference.md`

---

## 目录结构

```
atlhyper_web/
├── docs/                      # 开发文档
│   ├── ai-driven-development/ # AI 开发规范
│   └── task-management/       # 任务管理
├── src/                       # 源代码（待创建）
│   ├── app/                   # 页面路由
│   ├── api/                   # API 封装
│   ├── components/            # 组件
│   ├── store/                 # 状态管理
│   ├── types/                 # 类型定义
│   ├── i18n/                  # 国际化
│   ├── theme/                 # 主题系统
│   └── lib/                   # 工具函数
├── public/                    # 静态资源
│   └── logo.png               # geass 图标（淡绿蓝色）
├── CLAUDE.md                  # 本文件
└── package.json               # 待创建
```

---

## 开发流程

1. **开始前**: 读取本文件和 `03-progress.md`
2. **开发中**: 使用 TodoWrite 追踪任务
3. **完成后**: 更新 `03-progress.md` 和本文件
4. **每个 P 级别后**: 运行 `npm run build`

---

## 常用命令

```bash
# 开发
npm run dev

# 构建
npm run build

# 启动
npm run start
```

---

## 开发规范

### 模块化原则

1. **页面级模块化**: 每个页面独立目录，包含页面组件和专属子组件
2. **功能模块化**: 页面内功能拆分为独立组件
3. **通用组件复用**: 跨页面共享的组件放在 `components/common/`

### 代码行数限制

| 文件类型 | 最大行数 | 说明 |
|---------|---------|------|
| 页面文件 | 300 行 | 超出需拆分子组件 |
| 组件文件 | 200 行 | 超出需拆分 |
| 工具函数 | 150 行 | 超出需拆分模块 |

### 组件目录结构

```
src/components/
├── common/          # 通用基础组件
│   ├── PageHeader.tsx
│   ├── StatsCard.tsx
│   ├── DataTable.tsx
│   ├── StatusBadge.tsx
│   └── LoadingSpinner.tsx
├── layout/          # 布局组件
├── navigation/      # 导航组件
├── auth/            # 认证组件
└── cluster/         # 集群监控专用组件 (可选)
```

### 命名规范

| 类型 | 规范 | 示例 |
|------|------|------|
| 组件文件 | PascalCase | `PageHeader.tsx` |
| 工具函数 | camelCase | `formatDate.ts` |
| 类型文件 | camelCase | `cluster.ts` |
| 常量 | UPPER_SNAKE | `API_BASE_URL` |

---

## 注意事项

1. 每次完成任务后立即更新进度文档
2. 遵循 atlantis 代码风格
3. 所有代码必须有 TypeScript 类型
4. 禁止使用 `any`
5. 构建失败时先修复再继续
6. 浏览功能无需登录，操作功能需登录
7. 主题色为淡绿蓝色 (teal/cyan)
8. **单文件不超过 300 行**
9. **重复代码必须抽取为通用组件**

---

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2026-01-03 | 初始版本 |
| 1.1 | 2026-01-03 | 添加新需求：取消强制登录、国际化、主题、响应式、品牌色 |
