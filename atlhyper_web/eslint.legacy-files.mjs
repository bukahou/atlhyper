// 还带着 react-hooks 违规的旧文件。迁到数据层时一并修掉，修完从这里删除；⛔ 不得往里加新文件。
// 这些文件里的 react-hooks 规则暂时只报警告，其余文件一律按错误处理。
export const LEGACY_LINT_FILES = [
  "src/app/admin/deploy/components/useDeployPage.ts",
  "src/app/auth/callback/page.tsx",
  "src/app/settings/ai/components/BudgetConfigCard.tsx",
  "src/app/settings/ai/components/RoleOverviewCard.tsx",
  "src/app/settings/ai/components/UsageHistoryCard.tsx",
  "src/app/settings/ai/components/useAISettings.ts",
  "src/app/settings/ai/components/useProviderForm.ts",
  "src/app/settings/notif/page.tsx",
  "src/hooks/useAutoRefresh.ts",
];
