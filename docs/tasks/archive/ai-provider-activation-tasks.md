# AI Provider 激活机制优化 — 已完成

> 完成时间：2026-03-09

## 背景

移除全局 `ai_active_config.enabled` 开关和 `provider_id` "激活"概念。
新机制：角色分配即生效 — Provider 有角色就使用，无角色就不使用。

## 完成内容

### Phase 1: 后端重构 ✅

- `database/types.go`: `AIActiveConfig` → `AISettings`（仅保留 `ToolTimeout`）
- `database/interfaces.go`: Repository/Dialect 接口简化
- `database/sqlite/migrations.go`: `ai_active_config` 表 → `ai_settings` 表，删除所有 ALTER/DROP 迁移
- `database/sqlite/ai.go`: Dialect 实现更新
- `database/repo/ai_active.go` → `ai_settings.go`: 重写
- `database/sync.go`: `InitAIActiveConfig` → `InitAISettings`，删除 `MigrateOldAIConfig`
- `ai/role.go`: 移除 `active.Enabled` 检查
- `ai/service.go` + `ai/factory.go`: 字段更新
- `service/interfaces.go`: 方法签名更新
- `service/query/` + `service/operations/`: 实现更新
- `gateway/handler/admin/ai_provider.go`: 移除 ActiveConfig，新增 AISettings
- `gateway/routes.go`: `/api/v2/ai/active` → `/api/v2/ai/settings`
- `master.go`: 初始化流程更新

### Phase 2: 前端重构 ✅

- `api/ai-provider.ts`: `AIProvider` 移除 `isActive`，`ActiveConfig` → `AISettings`，API 端点更新
- `useAISettings.ts`: 移除 `globalEnabled`、`handleToggleEnabled`、`handleActivateProvider`
- `GlobalSettingsCard.tsx`: 移除启用开关，仅保留 Tool Timeout
- `ProviderCard.tsx`: 移除激活徽章、激活按钮、`onActivate` prop
- Chat `hooks.ts` + `types.ts` + `StatusViews.tsx`: 简化 AI 配置检查，移除 `not_enabled` 状态
- i18n 清理（zh.ts / ja.ts / types/i18n.ts）

### Phase 3: Provider 与角色分配解耦 ✅

- `ProviderModal.tsx`: 移除角色复选框，Provider 弹窗只负责模型配置
- `useProviderForm.ts`: 移除 `formRoles`、`toggleRole`
- `useAISettings.ts`: `handleSaveProvider` 移除 `updateProviderRoles` 调用
- `RoleOverviewCard.tsx`: 从只读改为可操作（Admin 可通过下拉选择器分配角色）
- `page.tsx`: 传递 providers 和 onRoleChanged 给 RoleOverviewCard

### Phase 4: 事件等级说明补全 ✅

- `BudgetConfigCard.tsx`: severity 下拉选项翻译 + 说明描述
- `IncidentList.tsx`: 表头和徽章增加 tooltip
- `incidents/page.tsx`: 状态过滤按钮增加 tooltip
- i18n 新增 severity/state 描述翻译键
