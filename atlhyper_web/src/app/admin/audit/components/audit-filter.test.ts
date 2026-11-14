import { describe, expect, it } from "vitest";
import type { AuditLogItem } from "@/types/auth";
import { filterAuditLogs } from "./audit-filter";

const now = Date.parse("2026-11-14T12:00:00Z");

function log(id: number, username: string, success: boolean, minutesAgo: number): AuditLogItem {
  return {
    id,
    timestamp: new Date(now - minutesAgo * 60_000).toISOString(),
    userId: id,
    username,
    role: 3,
    source: "web",
    action: "login",
    resource: "user",
    method: "POST",
    status: success ? 200 : 403,
    success,
    ip: "203.0.113.10",
    durationMs: 10,
  };
}

const logs = [log(1, "Admin", true, 5), log(2, "operator", false, 90), log(3, "viewer", true, 60 * 30)];
const everything = { timeRangeHours: 0, username: "", result: "all" as const };

describe("filterAuditLogs", () => {
  it("不限条件时全部保留", () => {
    expect(filterAuditLogs(logs, everything, now).map((entry) => entry.id)).toEqual([1, 2, 3]);
  });

  it("时间范围以传入的时刻为基准；0 表示不限", () => {
    expect(filterAuditLogs(logs, { ...everything, timeRangeHours: 1 }, now).map((entry) => entry.id)).toEqual([1]);
    expect(filterAuditLogs(logs, { ...everything, timeRangeHours: 24 }, now).map((entry) => entry.id)).toEqual([1, 2]);
  });

  it("用户名按包含匹配，不分大小写", () => {
    expect(filterAuditLogs(logs, { ...everything, username: "ADM" }, now).map((entry) => entry.id)).toEqual([1]);
    expect(filterAuditLogs(logs, { ...everything, username: "e" }, now).map((entry) => entry.id)).toEqual([2, 3]);
  });

  it("按结果筛选", () => {
    expect(filterAuditLogs(logs, { ...everything, result: "success" }, now).map((entry) => entry.id)).toEqual([1, 3]);
    expect(filterAuditLogs(logs, { ...everything, result: "failed" }, now).map((entry) => entry.id)).toEqual([2]);
  });
});
