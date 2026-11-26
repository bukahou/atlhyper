/**
 * 服务账号管理页的演示数据（没有 user.manage 权限时整页展示，均为虚构）
 *
 * 三个账号覆盖页面上的各种状态：启用中且有有效 / 已吊销的令牌、已停用且令牌已过期、启用中但还没有令牌。
 * 令牌前缀是占位符，不对应任何真实令牌。
 */
import type { ServiceAccountLimits, ServiceAccountView } from "@/api/service-accounts";

const MINUTE_MS = 60_000;
const DAY_MS = 24 * 60 * MINUTE_MS;

/** 相对现在的时刻：让「最近使用」「到期」在任何时候打开都合情合理 */
function relativeTime(offsetMs: number): string {
  return new Date(Date.now() + offsetMs).toISOString();
}

/** 演示用的业务参数（真实数据下由后端随列表下发） */
export const MOCK_SERVICE_ACCOUNT_LIMITS: ServiceAccountLimits = {
  assignableRoles: [0, 1, 2],
  maxActiveTokens: 10,
  maxExpiresInDays: 3650,
  defaultExpiresInDays: 90,
};

export const MOCK_SERVICE_ACCOUNTS: ServiceAccountView[] = [
  {
    id: 1,
    name: "demo-assistant",
    displayName: "Demo Voice Assistant",
    role: 0,
    enabled: true,
    createdAt: relativeTime(-30 * DAY_MS),
    lastUsedAt: relativeTime(-2 * MINUTE_MS),
    canIssueToken: true,
    tokens: [
      {
        id: 12,
        name: "demo-prod",
        tokenPrefix: "atlp_Dem",
        status: "active",
        createdBy: "demo-admin",
        createdAt: relativeTime(-10 * DAY_MS),
        expiresAt: relativeTime(80 * DAY_MS),
        lastUsedAt: relativeTime(-2 * MINUTE_MS),
        revokedAt: "",
      },
      {
        id: 11,
        name: "demo-initial",
        tokenPrefix: "atlp_Dem",
        status: "revoked",
        createdBy: "demo-admin",
        createdAt: relativeTime(-30 * DAY_MS),
        expiresAt: "",
        lastUsedAt: relativeTime(-10 * DAY_MS),
        revokedAt: relativeTime(-10 * DAY_MS),
      },
    ],
  },
  {
    id: 2,
    name: "demo-ci",
    displayName: "Demo CI",
    role: 1,
    enabled: false,
    createdAt: relativeTime(-120 * DAY_MS),
    lastUsedAt: relativeTime(-40 * DAY_MS),
    canIssueToken: true,
    tokens: [
      {
        id: 7,
        name: "demo-pipeline",
        tokenPrefix: "atlp_Dem",
        status: "expired",
        createdBy: "demo-admin",
        createdAt: relativeTime(-120 * DAY_MS),
        expiresAt: relativeTime(-30 * DAY_MS),
        lastUsedAt: relativeTime(-40 * DAY_MS),
        revokedAt: "",
      },
    ],
  },
  {
    id: 3,
    name: "demo-idle",
    displayName: "",
    role: 2,
    enabled: true,
    createdAt: relativeTime(-1 * DAY_MS),
    lastUsedAt: "",
    canIssueToken: true,
    tokens: [],
  },
];
