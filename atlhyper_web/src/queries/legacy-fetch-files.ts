/**
 * 还没迁到数据层的文件（仍在直接导入 @/api 或使用 useAutoRefresh）。
 *
 * 迁一个删一个，⛔ 不得往里加新文件。守卫测试 no-direct-fetch.test.ts 会检查：
 * 清单外的文件不得违规；清单里的文件一旦迁完必须从这里删掉。
 */
export const LEGACY_FETCH_FILES: string[] = [
  "app/admin/audit/page.tsx",
  "app/admin/commands/page.tsx",
  "app/admin/deploy/components/useDeployPage.ts",
  "app/admin/users/components/AddUserModal.tsx",
  "app/admin/users/components/DeleteUserModal.tsx",
  "app/admin/users/components/EditRoleModal.tsx",
  "app/admin/users/page.tsx",
  "app/aiops/chat/hooks.ts",
  "app/aiops/incidents/components/IncidentDetailModal.tsx",
  "app/aiops/incidents/page.tsx",
  "app/aiops/risk/components/TopEntities.tsx",
  "app/aiops/risk/page.tsx",
  "app/aiops/topology/components/NodeDetail.tsx",
  "app/aiops/topology/page.tsx",
  "app/auth/github/callback/page.tsx",
  "app/observe/apm/page.tsx",
  "app/observe/logs/components/LogsSummaryStrip.tsx",
  "app/observe/logs/page.tsx",
  "app/observe/slo/components/DomainDetail.tsx",
  "app/observe/slo/components/SLOTargetModal.tsx",
  "app/observe/slo/page.tsx",
  "app/settings/ai/components/BudgetConfigCard.tsx",
  "app/settings/ai/components/RoleOverviewCard.tsx",
  "app/settings/ai/components/UsageHistoryCard.tsx",
  "app/settings/ai/components/useAISettings.ts",
  "app/settings/github/components/useGitHubPage.ts",
  "app/settings/notif/page.tsx",
  "components/aiops/AIAnalysisSection.tsx",
  "components/aiops/AIReportCard.tsx",
  "components/auth/LoginDialog.tsx",
];
