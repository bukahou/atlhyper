# 前端数据层：统一用 TanStack Query 取数

> 状态：设计稿，待用户裁决「待裁决」一节后动工。
> 本设计只改前端，不新增也不修改任何后端接口，因此没有 API 定义与三层数据对照。

## 背景

前端没有数据层。72 个文件各自在 `useEffect` 里调接口，轮询靠 `useAutoRefresh`（22 处）或手写 `setInterval`（8 处）。
2026-11-14 盘点（`atlhyper_web/src`，39 个页面）得到的现状：

| 问题 | 现状 | 后果 |
|------|------|------|
| 请求不能取消，也不保证「后发的结果生效」 | `AbortController` 只在 AI 流式对话里有 1 处；轮询 10 秒一次，axios 超时 30 秒 | 慢请求的旧结果会盖掉新结果；切换筛选、连续点不同行时尤其明显 |
| 依赖数组写错 | 15 个集群资源页的 `fetchData` 依赖是 `[]`，闭包里的集群 ID 永远是首次渲染时的值 | 集群 ID 在程序内变化后，页面继续用旧 ID（或空串）轮询 |
| 权限没加载完就取数 | `/auth/me` 在 `Layout` 里异步加载，页面请求不等它；只有 `admin/users` 等了 | 9 个页面先闪一下演示数据；logs / apm / commands 首次请求被拦成空结果，要等下一轮轮询 |
| 每次切换路由重复请求 | 37 个页面各包一层 `Layout`，每次挂载都请求 `/auth/me` 与集群列表 | 无意义的重复请求 |
| 三态各写各的 | loading / error / empty 每页一套；「无集群」占位 7 处各写一份 | 4 个页面在没有集群时永远转圈；metrics 页的错误界面走不到 |
| 没有错误边界 | 没有 `error.tsx` / `global-error.tsx` | 任一组件渲染抛错，整页白屏 |
| 没有 lint | 没有 ESLint 配置与依赖；`lint` 脚本是 `next lint` | 依赖数组这类错误没有任何工具拦 |
| 分层约定没落实 | 规范写「页面只导入 datasource」，实际 49 个文件直接导入 `api/`；datasource 各文件返回形态不一 | 无法在一处统一加取消、缓存、权限 |

## 目标

- 所有读请求经 TanStack Query：同一份数据只请求一次、后发结果生效、卸载与参数变化时取消
- 权限与集群就绪之前不发请求、不渲染演示数据
- loading / error / empty / 无集群 四种状态用同一个组件
- 渲染抛错有兜底页面
- 依赖数组错误由 lint 拦住，绕过数据层的写法由守卫测试拦住

**不在范围内**：AI 对话的流式请求（SSE，保留现有实现）；后端接口；页面视觉。

## 核心架构

### 分层

```
页面 / 组件
   │  useQuery(xxxQuery(params))            只描述「要什么」
   ▼
src/queries/        [新增]  queryOptions 工厂：键、取数函数、轮询间隔、启用条件
   │
   ▼
src/datasource/     [整理]  mock / 真实接口切换；统一返回解包后的数据；接收 AbortSignal
   │
   ▼
src/api/            [不变]  axios 封装与各资源接口
```

页面与组件只导入 `queries/`（读）与 `datasource/`（写操作）。不再直接导入 `api/` 的函数（类型除外）。

### 查询键

键由工厂函数生成，页面不手写数组。形状固定为：

```ts
[域, 资源, { clusterId?, mode, ...筛选参数 }]
// 例：["cluster", "events", { clusterId: "c1", mode: "api" }]
```

- `clusterId` 进键：切换集群即切换缓存，旧集群的数据不会显示在新集群下
- `mode`（`api` / `mock`）进键：真实数据与演示数据永不共用缓存

### 取数描述（queryOptions 工厂）

```ts
// src/queries/cluster.ts
export function eventOverviewQuery(clusterId: string) {
  return queryOptions({
    queryKey: clusterKeys.events(clusterId),
    queryFn: ({ signal }) => datasource.cluster.getEventOverview({ ClusterID: clusterId }, signal),
    refetchInterval: REFRESH.standard,
    enabled: clusterId !== "",
  });
}

// 页面
const { clusterId } = useCurrentCluster();
const query = useQuery(eventOverviewQuery(clusterId));
return <QueryView query={query} empty={...}>{(data) => <EventTable data={data} />}</QueryView>;
```

工厂是纯函数，可以不依赖 React 单测（`QueryClient.fetchQuery`）。

### 全局默认值

| 项 | 值 | 理由 |
|----|----|------|
| 轮询间隔 | `REFRESH.fast` 10 秒 / `REFRESH.standard` 沿用 `NEXT_PUBLIC_REFRESH_INTERVAL` / `REFRESH.slow` 30 秒 | 各页保持现在的间隔，集中成常量 |
| `refetchIntervalInBackground` | `false` | 标签页不可见时停止轮询，减轻后端与 ClickHouse 负载 |
| `refetchOnWindowFocus` | `true` | 切回标签页立即刷新一次 |
| `retry` | 读请求 1 次；401 / 403 不重试 | 鉴权错误交给现有的 `authErrorManager` |
| `staleTime` | 轮询查询 = 轮询间隔；详情类 0 | 同一页面多个组件用同一份数据时不重复请求 |
| 上一次请求未返回 | 不发下一次（库的默认行为） | 现在 10 秒轮询叠 30 秒超时会并发 |

### 会话与权限

- `/auth/me` 与集群列表从 `Layout` 提到根部的 `SessionLoader`（挂在 `app/providers.tsx`），全站只加载一次；登录、登出、403 时照旧触发刷新
- `authStore`（token、权限、`meLoaded`）保留，`useCan` 不变
- 新增 `useSessionReady()`：`meLoaded` 为真才算就绪。需要权限的查询写 `enabled: ready && can("xxx")`
- datasource 里的 `canNow` 把关删除，改由查询的 `enabled` 把关（`canNow` 不是响应式的，是首次请求被拦成空结果的原因）
- 纯管理页（审计、通知、AI 设置等）：就绪后才决定显示真实数据还是演示数据，就绪前显示加载中

### 集群

- `useCurrentCluster()` 返回 `{ clusterId, ready }`；集群列表加载完且已选定集群才 `ready`
- 所有按集群取数的工厂把 `clusterId === ""` 作为不启用条件
- 最后一期去掉切换集群时的整页刷新：键里带集群，切换即换数据

### 四态组件

```tsx
<QueryView query={query} empty={<EmptyState ... />} requireCluster>
  {(data) => ...}
</QueryView>
```

| 状态 | 条件 | 显示 |
|------|------|------|
| 无集群 | `requireCluster` 且没有选定集群 | 统一的「未接入集群」占位 |
| 加载中 | 首次加载、还没有数据 | 统一的加载指示 |
| 出错 | 没有数据且请求失败 | 错误文案 + 重试按钮 |
| 刷新失败 | 已有数据、后台刷新失败 | 继续显示旧数据，角落提示「刷新失败」 |
| 空 | 请求成功、数据为空 | `empty` 传入的空状态（符合「组件禁止隐藏」的规范） |

判定顺序固定为上表从上到下，「无集群」先于「加载中」。

### 错误边界

- `app/error.tsx`：路由段内渲染抛错的兜底，带「重试」
- `app/global-error.tsx`：根布局抛错的兜底
- 请求失败不抛给边界，由 `QueryView` 就地显示

### 写操作

- 用 `useMutation`；成功后 `invalidateQueries` 让相关列表与详情重新取数
- 指令是异步执行的（入队后由 Agent 执行）。现有的「2 秒后再刷新一次」改为 `useDelayedInvalidate`，卸载时清理定时器

### Lint 与守卫

- 新增 `eslint.config.mjs`：`eslint-config-next`（含 `react-hooks`）+ `@tanstack/eslint-plugin-query`
- `react-hooks/exhaustive-deps` 设为 error
- `lint` 脚本改为 `eslint .`（`next lint` 在 Next 16 已移除；本机尚未实跑确认）
- 守卫测试 `src/queries/no-direct-fetch.test.ts`（TypeScript 语法树，沿用 `no-raw-html.test.ts` 的做法）：
  - `app/` 与 `components/` 下不得值导入 `@/api/*`
  - 不得调用 `useAutoRefresh`；不得在 `useEffect` 里调 datasource 的读函数
  - 迁移期间用「尚未迁移」清单豁免，每迁一批删一批，最后一期清空

## 功能一：基础设施与样板页（第 1 期）

**用户故事**：打开角色页与事件页，行为和现在一样；切换路由不再重复请求会话信息；页面渲染出错时看到兜底页而不是白屏。

**前端实现**

- `src/queries/client.ts` [新增]：`QueryClient` 与全局默认值
- `src/queries/keys.ts` [新增]：键工厂
- `src/queries/refresh.ts` [新增]：轮询间隔常量
- `src/queries/admin.ts`、`src/queries/cluster.ts` [新增]：样板页用到的工厂
- `src/app/providers.tsx` [修改]：挂 `QueryClientProvider` 与 `SessionLoader`
- `src/components/session/SessionLoader.tsx` [新增]；`src/components/layout/Layout.tsx` [修改]：去掉会话与集群列表的加载
- `src/hooks/useSessionReady.ts`、`src/hooks/useCurrentCluster.ts` [新增]
- `src/components/common/QueryView.tsx` [新增]
- `src/app/error.tsx`、`src/app/global-error.tsx` [新增]
- `eslint.config.mjs` [新增]；`package.json` [修改]
- 样板：`app/admin/roles/page.tsx`（最小的单查询页）、`app/cluster/event/page.tsx`（集群页通用写法）

**验收**：两个样板页正常；浏览器网络面板里切换路由不再出现 `/auth/me` 与集群列表请求；人为让一个组件抛错能看到兜底页。

## 功能二：集群资源页与详情弹窗（第 2 期）

**用户故事**：17 个集群资源页与 17 个详情弹窗的数据始终属于当前集群；连续点不同行时弹窗显示的是最后点的那一行。

- 17 个 `app/cluster/*/page.tsx`、17 个 `components/*/XxxDetailModal.tsx`、`ResourceEventsTab`、`PodLogsViewer`
- `datasource/cluster.ts`：43 个函数统一返回解包后的数据、接收 `signal`
- 重启 / 扩缩容 / cordon 等写操作改 `useMutation`

**一并修复**：15 个页面的依赖数组错误；4 处卸载后仍会触发的延时刷新。

## 功能三：可观测性页面（第 3 期）

**用户故事**：指标、链路、日志、SLO 四个页面在切换时间范围、服务、筛选条件时不再出现旧结果覆盖新结果；有权限的用户首屏就能看到日志与链路。

- `app/observe/{metrics,apm,logs,slo}` 及其子组件；`useSignalFreshness`（四页共用一份查询）
- `datasource/{metrics,apm,logs,slo}.ts` 统一形态；`apm.ts` 不再把错误吞成空数组

**一并修复**：metrics 的错误界面走不到、top 节点历史每 10 秒整批重拉；logs 首屏重复请求；slo 页自己取第一个集群而不看当前集群；apm 的链路详情没有错误处理。

## 功能四：其余页面与收尾（第 4 期）

**用户故事**：全站取数方式一致；切换集群不再整页刷新；没有集群时每个页面都显示同一个占位。

- `app/overview`、`app/aiops/{risk,incidents,topology}`、`app/admin/{users,audit,commands,deploy}`、`app/settings/{ai,notif,github}`
- 删除 `useAutoRefresh`、未被使用的 `useApiCall`
- 去掉集群切换时的 `window.location.reload()`
- 守卫测试的豁免清单清空

**一并修复**：risk / topology / incidents / metrics 没有集群时永远转圈；9 个管理页先闪演示数据；settings/ai 页重复请求 Provider 列表。

## 行为变化（用户可感知）

- 标签页不可见时停止轮询，切回时立即刷新
- 后台刷新失败时继续显示上一次的数据，并提示刷新失败（现在有的页面静默、有的页面整页报错）
- 切换集群不再整页刷新（第 4 期起）
- 管理页不再先闪一下演示数据

## 实施阶段

```
第 1 期 基础设施 + 2 个样板 ──► 第 2 期 集群页 34 个文件 ──► 第 3 期 可观测性 ──► 第 4 期 其余 + 收尾
```

每期结束：`tsc`、`vitest`、`eslint`、`next build` 通过，推 dev，在浏览器里逐页核对。
迁移期间新旧两种写法并存，由守卫测试的豁免清单标明哪些文件还没迁。

## 待裁决

**1. 新增依赖并在项目里执行 `npm install`**

| | 包 | 用途 |
|---|---|---|
| 运行时 | `@tanstack/react-query` v5 | 数据层 |
| 开发 | `eslint`、`eslint-config-next`、`@tanstack/eslint-plugin-query` | lint |

`package.json` 与 `package-lock.json` 会变；CI 用 `npm ci --legacy-peer-deps` 安装。
不装 jsdom / testing-library：工厂是纯函数，在现有的 node 测试环境里就能测。

**2. 迁移范围**：四期全做，还是只做第 1、2 期。

**3. lint 是否进 CI 并作为必过项**。

## 文件变更清单

```
atlhyper_web/
├── eslint.config.mjs                         [新增]
├── package.json / package-lock.json          [修改]
└── src/
    ├── queries/                              [新增] client / keys / refresh / 各域工厂 / 守卫测试
    ├── hooks/
    │   ├── useSessionReady.ts                [新增]
    │   ├── useCurrentCluster.ts              [新增]
    │   ├── useDelayedInvalidate.ts           [新增]
    │   ├── useAutoRefresh.ts                 [第 4 期删除]
    │   └── useAuthError.ts                   [修改] 删除未使用的 useApiCall
    ├── components/
    │   ├── common/QueryView.tsx              [新增]
    │   ├── session/SessionLoader.tsx         [新增]
    │   └── layout/Layout.tsx                 [修改]
    ├── datasource/*.ts                       [修改] 统一返回形态、接收 signal、去掉 canNow
    ├── app/
    │   ├── providers.tsx                     [修改]
    │   ├── error.tsx / global-error.tsx      [新增]
    │   └── **/page.tsx 与各组件              [按期修改，约 72 个文件]
.github/workflows/test.yml                    [修改] 增加 lint 步骤（视裁决 3）
```
