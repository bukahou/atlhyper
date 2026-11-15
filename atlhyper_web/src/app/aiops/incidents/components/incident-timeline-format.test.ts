import { describe, expect, it } from "vitest";
import { jaIncidentDetail } from "@/i18n/locales/incident-ja";
import { zhIncidentDetail } from "@/i18n/locales/incident-zh";
import type { IncidentTimelineItem, IncidentTimelineKind } from "@/types/incident-view";
import { timelineLine, timelineSummary } from "./incident-timeline-format";

const SEVERITY_ZH = { low: "低", medium: "中", high: "高", critical: "严重" };
const AT = new Date(2026, 10, 15, 14, 53).toISOString();
const LAST = new Date(2026, 10, 15, 15, 0).toISOString();

describe("timelineSummary", () => {
  it("非检测条目算变化，检测条目按合并条数累加", () => {
    const items: IncidentTimelineItem[] = [
      { at: AT, kind: "created", risk: 58, severity: "medium" },
      { at: AT, kind: "severity_raised", risk: 77, severity: "high" },
      { at: AT, kind: "escalated", risk: 88 },
      { at: AT, kind: "anomaly", count: 5, firstAt: AT, lastAt: LAST },
      { at: AT, kind: "anomaly", count: 2, firstAt: AT, lastAt: LAST },
    ];
    expect(timelineSummary(items)).toEqual({ changes: 3, merged: 7 });
  });

  it("空列表", () => {
    expect(timelineSummary([])).toEqual({ changes: 0, merged: 0 });
  });
});

describe("timelineLine", () => {
  it("开立事件：带严重度与风险", () => {
    expect(timelineLine({ at: AT, kind: "created", risk: 58, severity: "medium" }, zhIncidentDetail, SEVERITY_ZH)).toEqual({
      label: "开立事件",
      notes: ["严重度 中", "风险 58"],
      merged: false,
    });
  });

  it("严重度升高：升到哪一档写进主文案", () => {
    expect(
      timelineLine({ at: AT, kind: "severity_raised", risk: 77.4, severity: "high" }, zhIncidentDetail, SEVERITY_ZH),
    ).toEqual({ label: "严重度升到 高", notes: ["风险 77"], merged: false });
  });

  it("升级：没有严重度字段，只有风险", () => {
    expect(timelineLine({ at: AT, kind: "escalated", risk: 88 }, zhIncidentDetail, SEVERITY_ZH)).toEqual({
      label: "升级为事件中",
      notes: ["风险 88"],
      merged: false,
    });
  });

  it("风险为 null 或缺省都不写", () => {
    expect(timelineLine({ at: AT, kind: "closed", risk: null }, zhIncidentDetail, SEVERITY_ZH).notes).toEqual([]);
    expect(timelineLine({ at: AT, kind: "closed_stale" }, zhIncidentDetail, SEVERITY_ZH).label).toBe("主体失联，事件关闭");
  });

  it("合并的检测：次数与时间范围", () => {
    expect(
      timelineLine({ at: AT, kind: "anomaly", count: 5, firstAt: AT, lastAt: LAST }, zhIncidentDetail, SEVERITY_ZH),
    ).toEqual({
      label: "异常检测",
      notes: ["× 5（11/15 14:53 – 11/15 15:00），已汇总到上方「关键证据」"],
      merged: true,
    });
  });

  it("不出现内部写法：没见过的 kind 给通用文案，没见过的严重度原样", () => {
    const line = timelineLine(
      { at: AT, kind: "something_new" as IncidentTimelineKind, severity: "extreme" },
      zhIncidentDetail,
      SEVERITY_ZH,
    );
    expect(line.label).toBe("其他变化");
    expect(line.notes).toEqual(["严重度 extreme"]);
  });

  it("每种 kind 在中日文里都有文案", () => {
    const kinds: IncidentTimelineKind[] = [
      "created", "escalated", "severity_raised", "recovery_started", "recurred", "closed", "closed_stale", "anomaly",
    ];
    for (const texts of [zhIncidentDetail, jaIncidentDetail]) {
      for (const kind of kinds) {
        const line = timelineLine({ at: AT, kind, risk: 50, severity: "high", count: 1 }, texts, SEVERITY_ZH);
        expect(line.label).toBeTruthy();
        expect(line.label).not.toBe(texts.timelineUnknownKind);
        expect([line.label, ...line.notes].join(" ")).not.toMatch(/\{\w+\}/);
      }
    }
  });
});
