import { describe, expect, it } from "vitest";
import { INCIDENT_SEEDS } from "./incidents";
import { mockIncidentDetail } from "./incident-detail";

/** 演示数据要守住契约（功能四）的约定，界面才能拿它当真实响应开发 */
describe("mockIncidentDetail 与契约一致", () => {
  const details = INCIDENT_SEEDS.map((seed) => mockIncidentDetail(seed.incident.id));

  it("每个事件都带上新增的五项", () => {
    for (const detail of details) {
      expect(detail).toHaveProperty("headline");
      expect(detail.impact).toBeDefined();
      expect(Array.isArray(detail.evidence)).toBe(true);
      expect(Array.isArray(detail.members)).toBe(true);
      expect(Array.isArray(detail.timelineView)).toBe(true);
    }
  });

  it("members 与 entities 一一对应，恰有一个 isLead 且就是 rootCause", () => {
    for (const detail of details) {
      expect(detail.members.map((member) => member.entityKey)).toEqual(detail.entities.map((entity) => entity.entityKey));
      const leads = detail.members.filter((member) => member.isLead);
      expect(leads).toHaveLength(1);
      expect(leads[0].entityKey).toBe(detail.rootCause);
    }
  });

  it("present: false 的成员，状态三项为空", () => {
    for (const member of details.flatMap((detail) => detail.members).filter((candidate) => !candidate.present)) {
      expect(member.status).toBe("");
      expect(member.restarts).toBeNull();
      expect(member.ready).toBe("");
    }
  });

  it("headline 是 evidence 里的第一条（分数最高）；没有证据时为 null", () => {
    for (const detail of details) {
      if (detail.evidence.length === 0) {
        expect(detail.headline).toBeNull();
        continue;
      }
      expect(detail.headline).toEqual({
        signal: detail.evidence[0].signal,
        reason: detail.evidence[0].reason,
        affectedMembers: detail.evidence[0].affectedMembers,
        totalMembers: detail.members.length,
      });
    }
  });

  it("证据按信号去重", () => {
    for (const detail of details) {
      const signals = detail.evidence.map((item) => item.signal);
      expect(new Set(signals).size).toBe(signals.length);
    }
  });

  it("覆盖界面的几种分支：全部中招、部分中招、改版前没有证据、只有依赖传播", () => {
    const headlines = details.map((detail) => detail.headline);
    expect(headlines.some((h) => h !== null && h.totalMembers > 1 && h.affectedMembers === h.totalMembers)).toBe(true);
    expect(headlines.some((h) => h !== null && h.affectedMembers < h.totalMembers)).toBe(true);
    const noEvidence = details.filter((detail) => detail.evidence.length === 0);
    expect(noEvidence.some((detail) => detail.timelineView.some((item) => item.kind === "anomaly"))).toBe(true);
    expect(noEvidence.some((detail) => !detail.timelineView.some((item) => item.kind === "anomaly"))).toBe(true);
  });

  it("风险是 0–100 的整数；影响数字只对有副本的工作负载给出", () => {
    for (const detail of details) {
      for (const item of detail.timelineView) {
        if (item.risk !== undefined && item.risk !== null) {
          expect(Number.isInteger(item.risk)).toBe(true);
          expect(item.risk).toBeGreaterThanOrEqual(0);
          expect(item.risk).toBeLessThanOrEqual(100);
        }
      }
      const hasReplicas = ["deployment", "statefulset", "daemonset"].includes(detail.subjectType);
      if (!hasReplicas) expect(detail.impact.desiredReplicas).toBeNull();
    }
  });
});
