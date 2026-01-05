# AtlHyper Web - 总任务列表

## 概述

本文档记录 atlhyper_web 前端重构项目的所有任务，按优先级和阶段组织。

**最后更新**: 2026-01-03 (v1.1 - 添加新需求)

---

## 任务统计

| 阶段 | 总任务 | 已完成 | 进行中 | 待开始 |
|------|--------|--------|--------|--------|
| Phase 0: 初始化 | 10 | 0 | 0 | 10 |
| Phase 1: P0 核心 | 16 | 0 | 0 | 16 |
| Phase 2: P1 监控 | 8 | 0 | 0 | 8 |
| Phase 3: P2 扩展 | 8 | 0 | 0 | 8 |
| Phase 4: P3 系统 | 6 | 0 | 0 | 6 |
| **合计** | **48** | **0** | **0** | **48** |

---

## Phase 0: 项目初始化

**优先级**: 最高
**任务数**: 10

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 0.1 | 创建 Next.js 16 项目 | 待开始 | 无 | package.json, 基础结构 |
| 0.2 | 配置 TypeScript | 待开始 | 0.1 | tsconfig.json |
| 0.3 | 配置 Tailwind CSS v4 | 待开始 | 0.1 | tailwind.config.ts, globals.css |
| 0.4 | 创建目录结构 | 待开始 | 0.1 | src/api/, src/components/, src/i18n/, src/theme/ |
| 0.5 | 创建通用类型定义 | 待开始 | 0.2 | src/types/common.ts, src/types/i18n.ts |
| 0.6 | 配置 Axios 请求封装 | 待开始 | 0.5 | src/api/core/request.ts |
| 0.7 | 复制 geass 图标资源 | 待开始 | 0.1 | public/logo.png (淡绿蓝色) |
| 0.8 | 创建 CLAUDE.md | 待开始 | 0.4 | CLAUDE.md |
| 0.9 | 配置品牌主题色 (teal/cyan) | 待开始 | 0.3 | globals.css 主题变量 |
| 0.10 | 验证基础构建 | 待开始 | 0.1-0.9 | npm run build 成功 |

---

## Phase 1: P0 核心功能

**优先级**: P0 - 阻塞性
**任务数**: 16

### 1.1 国际化 (FR-014)

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 1.1.1 | I18n 类型定义 | 待开始 | 0.10 | src/types/i18n.ts |
| 1.1.2 | 中文语言包 | 待开始 | 1.1.1 | src/i18n/locales/zh.ts |
| 1.1.3 | 日文语言包 | 待开始 | 1.1.1 | src/i18n/locales/ja.ts |
| 1.1.4 | I18nProvider 实现 | 待开始 | 1.1.2, 1.1.3 | src/i18n/context.tsx |
| 1.1.5 | LanguageSwitcher 组件 | 待开始 | 1.1.4 | src/components/navigation/LanguageSwitcher.tsx |

### 1.2 主题系统 (FR-015)

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 1.2.1 | ThemeProvider 实现 | 待开始 | 0.10 | src/theme/context.tsx |
| 1.2.2 | ThemeSwitcher 组件 | 待开始 | 1.2.1 | src/components/navigation/ThemeSwitcher.tsx |

### 1.3 布局框架 (FR-002, FR-016)

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 1.3.1 | Layout 布局容器 | 待开始 | 1.1.4, 1.2.1 | src/components/layout/Layout.tsx |
| 1.3.2 | Sidebar 组件 | 待开始 | 1.3.1 | src/components/navigation/Sidebar.tsx |
| 1.3.3 | Navbar 组件 | 待开始 | 1.3.1, 1.1.5, 1.2.2 | src/components/navigation/Navbar.tsx |
| 1.3.4 | MobileMenu 组件 | 待开始 | 1.3.1 | src/components/navigation/MobileMenu.tsx |
| 1.3.5 | 响应式断点适配 | 待开始 | 1.3.1-1.3.4 | 移动端/平板/桌面端布局 |

### 1.4 认证系统 (FR-001 - 操作级别)

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 1.4.1 | Auth 类型定义 | 待开始 | 0.10 | src/types/auth.ts |
| 1.4.2 | Auth API 封装 | 待开始 | 0.6, 1.4.1 | src/api/auth.ts |
| 1.4.3 | AuthStore 实现 | 待开始 | 1.4.1 | src/store/authStore.ts |
| 1.4.4 | LoginDialog 组件 | 待开始 | 1.4.2, 1.4.3 | src/components/auth/LoginDialog.tsx |
| 1.4.5 | AuthGuard 操作守卫 | 待开始 | 1.4.3, 1.4.4 | src/components/auth/AuthGuard.tsx |
| 1.4.6 | UserMenu 组件 | 待开始 | 1.4.3 | src/components/navigation/UserMenu.tsx |

### 1.5 P0 验证

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 1.5.1 | 验证 P0 构建 | 待开始 | 1.1-1.4 | npm run build 成功 |

---

## Phase 2: P1 核心监控

**优先级**: P1 - 核心功能
**任务数**: 8

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 2.1 | Cluster 类型定义 | 待开始 | 1.5.1 | src/types/cluster.ts, pod.ts, node.ts |
| 2.2 | Pod/Node API 封装 | 待开始 | 2.1 | src/api/pod.ts, src/api/node.ts |
| 2.3 | ClusterStore 实现 | 待开始 | 2.1 | src/store/clusterStore.ts |
| 2.4 | Analysis 页面 (FR-003) | 待开始 | 2.2, 2.3 | src/app/(dashboard)/analysis/page.tsx |
| 2.5 | Workbench 页面 (FR-004) | 待开始 | 2.2, 1.4.5 | src/app/(dashboard)/workbench/page.tsx |
| 2.6 | Pod 监控页面 (FR-005) | 待开始 | 2.2, 1.4.5 | src/app/(dashboard)/cluster/pod/page.tsx |
| 2.7 | Node 监控页面 (FR-006) | 待开始 | 2.2, 1.4.5 | src/app/(dashboard)/cluster/node/page.tsx |
| 2.8 | 验证 P1 构建 | 待开始 | 2.1-2.7 | npm run build 成功 |

---

## Phase 3: P2 扩展监控

**优先级**: P2 - 增强功能
**任务数**: 8

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 3.1 | Deployment API 封装 | 待开始 | 2.8 | src/api/deployment.ts |
| 3.2 | Service API 封装 | 待开始 | 2.8 | src/api/service.ts |
| 3.3 | Deployment 页面 (FR-007) | 待开始 | 3.1 | src/app/(dashboard)/cluster/deployment/page.tsx |
| 3.4 | Service 页面 (FR-008) | 待开始 | 3.2 | src/app/(dashboard)/cluster/service/page.tsx |
| 3.5 | Namespace 页面 (FR-009) | 待开始 | 2.8 | src/app/(dashboard)/cluster/namespace/page.tsx |
| 3.6 | Ingress 页面 (FR-010) | 待开始 | 2.8 | src/app/(dashboard)/cluster/ingress/page.tsx |
| 3.7 | Alert 告警页面 (FR-011) | 待开始 | 2.8 | src/app/(dashboard)/cluster/alert/page.tsx |
| 3.8 | 验证 P2 构建 | 待开始 | 3.1-3.7 | npm run build 成功 |

---

## Phase 4: P3 系统管理

**优先级**: P3 - 优化项
**任务数**: 6

| # | 任务 | 状态 | 依赖 | 产出 |
|---|------|------|------|------|
| 4.1 | Metrics API 封装 | 待开始 | 3.8 | src/api/metrics.ts |
| 4.2 | Metrics 页面 (FR-012) | 待开始 | 4.1 | src/app/(dashboard)/system/metrics/page.tsx |
| 4.3 | Logs 页面 | 待开始 | 3.8 | src/app/(dashboard)/system/logs/page.tsx |
| 4.4 | User 管理页面 (FR-013) | 待开始 | 3.8 | src/app/(dashboard)/system/users/page.tsx |
| 4.5 | Audit 审计页面 | 待开始 | 3.8 | src/app/(dashboard)/system/audit/page.tsx |
| 4.6 | 最终验证 | 待开始 | 4.1-4.5 | 全功能测试通过 |

---

## 新需求汇总 (v1.1)

### 访问控制变更

| 原设计 | 新设计 |
|--------|--------|
| 强制登录后才能访问 | 浏览无需登录，操作时登录 |
| 登录页面 | 登录对话框 |
| 路由级别守卫 | 操作级别守卫 |

### 新增功能

| FR | 功能 | 任务 |
|-----|------|------|
| FR-014 | 国际化支持 (中日双语) | 1.1.1-1.1.5 |
| FR-015 | 主题切换 (亮/暗) | 1.2.1-1.2.2 |
| FR-016 | 响应式布局 | 1.3.5 |

### 品牌标识

| 项目 | 图标 | 主题色 |
|------|------|--------|
| Geass | geass | 淡紫色 |
| Atlantis | geass | 淡蓝色 |
| **AtlHyper** | **geass** | **淡绿蓝色 (teal/cyan)** |

---

## 任务状态说明

| 状态 | 含义 |
|------|------|
| 待开始 | 任务尚未开始 |
| 进行中 | 正在处理 |
| 已完成 | 任务完成 |
| 阻塞 | 遇到阻塞问题 |
| 跳过 | 任务跳过不做 |

---

## 当前焦点

**当前阶段**: Phase 0 - 项目初始化
**当前任务**: 待开始 (0.1 创建 Next.js 16 项目)
**下一步**: 初始化项目结构

---

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2026-01-03 | 初始版本，规划 40 个任务 |
| 1.1 | 2026-01-03 | 添加新需求，任务增至 48 个 |
