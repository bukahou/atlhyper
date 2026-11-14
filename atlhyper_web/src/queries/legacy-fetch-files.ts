/**
 * 还没迁到数据层的文件（仍在直接导入 @/api 或使用 useAutoRefresh）。
 *
 * 迁一个删一个，⛔ 不得往里加新文件。守卫测试 no-direct-fetch.test.ts 会检查：
 * 清单外的文件不得违规；清单里的文件一旦迁完必须从这里删掉。
 */
export const LEGACY_FETCH_FILES: string[] = [
  "app/admin/deploy/components/useDeployPage.ts",
  "app/auth/github/callback/page.tsx",
  "app/settings/ai/components/BudgetConfigCard.tsx",
  "app/settings/ai/components/RoleOverviewCard.tsx",
  "app/settings/ai/components/UsageHistoryCard.tsx",
  "app/settings/ai/components/useAISettings.ts",
  "app/settings/github/components/useGitHubPage.ts",
  "app/settings/notif/page.tsx",
  "components/auth/LoginDialog.tsx",
];
