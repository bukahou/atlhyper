# AtlHyper Web - AI 驱动开发指南

## 概述

本文档定义 atlhyper_web 前端重构项目的 AI 驱动开发规范，旨在指导 AI 工具进行高效、可控的开发工作。

---

## 硬性要求

### 绝对禁止事项

1. **禁止修改现有代码**
   - 不得修改 `/home/wuxiafeng/AtlHyper/GitHub/atlhyper/web/` 目录下的任何文件
   - 不得修改后端代码（atlhyper_master, atlhyper_agent 等）
   - 只能复制、参考现有代码，不能直接编辑

2. **原因说明**
   - 一旦修改现存代码，开发失败后将无法回滚到之前的版本
   - 会导致项目无法继续进行
   - 新前端需要完全独立开发

3. **正确做法**
   - 所有新代码写入 `atlhyper_web/` 目录
   - 参考现有前端的 API 调用方式和数据结构
   - 复制必要的类型定义和接口文档

---

## 项目定位

atlhyper_web 是 AtlHyper 项目的新一代前端，采用现代技术栈重构，实现前后端完全解耦。

### 技术栈

| 类别 | 技术 | 版本 |
|------|------|------|
| 框架 | Next.js | 16.x |
| UI库 | React | 19.x |
| 语言 | TypeScript | 5.x |
| 样式 | Tailwind CSS | 4.x |
| 状态管理 | Zustand | 5.x |
| 图标 | Lucide React | latest |
| HTTP 客户端 | axios | latest |

### 参考项目

| 项目 | 参考内容 |
|------|---------|
| Geass | AI 驱动开发规范、文档结构 |
| atlantis | 代码风格、组件设计、类型定义 |
| atlhyper/web | API 接口、业务逻辑、页面结构 |

---

## 目录结构

```
atlhyper_web/
├── docs/                          # 开发文档
│   ├── ai-driven-development/     # AI 开发规范
│   │   ├── 00-development-guide.md    # 本文档
│   │   ├── 01-spec.md                 # 功能规格书
│   │   ├── 02-api-reference.md        # API 参考
│   │   └── 03-progress.md             # 开发进度
│   └── task-management/           # 任务管理
│       ├── master-task-list.md        # 总任务列表
│       └── phase-{n}/                 # 各阶段任务
├── src/
│   ├── app/                       # Next.js App Router
│   ├── api/                       # API 层封装
│   ├── components/                # UI 组件
│   ├── store/                     # Zustand 状态管理
│   ├── types/                     # TypeScript 类型定义
│   ├── lib/                       # 工具函数
│   └── styles/                    # 全局样式
├── public/                        # 静态资源
├── package.json
├── tsconfig.json
├── next.config.ts
└── CLAUDE.md                      # AI 上下文文件
```

---

## 开发流程

### 阶段 1: 解析需求

1. 读取现有前端的路由配置 (`web/src/router/index.js`)
2. 分析 API 接口 (`web/src/api/*.js`)
3. 确定功能需求和优先级

### 阶段 2: 规划任务

1. 按优先级 P0 → P1 → P2 → P3 排序
2. 将功能分解为具体任务
3. 使用 TodoWrite 创建任务清单
4. 更新 `master-task-list.md`

### 阶段 3: 实现功能

**执行顺序：**
```
1. 类型定义 (types/)
   └── 数据模型、请求/响应类型

2. API 层 (api/)
   └── API 函数封装

3. 状态管理 (store/)
   └── Zustand Store

4. 组件 (components/)
   └── UI 组件实现

5. 页面 (app/)
   └── 页面路由和逻辑
```

### 阶段 4: 验证构建

每完成一个 P 级别后：
```bash
npm run build
```

---

## 任务状态管理

### 状态定义

| 状态 | 含义 |
|------|------|
| pending | 任务尚未开始 |
| in_progress | 正在处理（同时只能一个） |
| completed | 任务完成 |
| blocked | 遇到阻塞 |

### 进度更新

- 每完成一个任务立即更新 `03-progress.md`
- 每个 P 级别完成后生成报告
- 记录所有创建/修改的文件

---

## 代码规范

### 遵循 atlantis 代码风格

1. **文件命名**
   - 组件：PascalCase (`Navbar.tsx`)
   - 工具函数：camelCase (`content.ts`)
   - 类型定义：camelCase (`content.ts`)

2. **组件结构**
   ```typescript
   "use client"; // 客户端组件必须

   import { ... } from "...";

   interface Props {
     // 明确的 Props 类型
   }

   export function ComponentName({ ...props }: Props) {
     // 组件逻辑
   }
   ```

3. **类型定义**
   - 所有代码必须有完整的 TypeScript 类型
   - 禁止使用 `any`
   - 接口使用 `interface`，联合类型使用 `type`

4. **样式**
   - 使用 Tailwind CSS 类
   - 保持响应式设计
   - 支持暗色模式

---

## 上下文管理

### 上下文有限问题

由于 AI 的上下文有限，必须：

1. **及时更新进度文档**
   - 每完成一个功能立即记录
   - 详细记录创建的文件和修改内容

2. **保持文档同步**
   - `master-task-list.md` 反映真实进度
   - `CLAUDE.md` 包含最新上下文

3. **切换 Chat 时**
   - 首先读取 `CLAUDE.md`
   - 查看 `03-progress.md` 了解当前状态
   - 继续未完成的任务

---

## 质量检查清单

### 每个任务完成时

- [ ] 代码有完整的 TypeScript 类型定义
- [ ] 遵循项目代码风格
- [ ] 必要处添加注释
- [ ] 无 console.log 遗留

### 每个 P 级别完成时

- [ ] `npm run build` 成功
- [ ] 功能可正常使用
- [ ] 无 TypeScript 错误
- [ ] 更新进度文档

---

## 版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0 | 2026-01-03 | 初始版本 |
