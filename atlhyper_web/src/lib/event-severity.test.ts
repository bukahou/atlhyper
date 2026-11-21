import { describe, expect, it } from "vitest";
import { eventSeverityBadge, eventSeverityLabel, matchesSeverityFilter } from "./event-severity";

const t = { criticalEvents: "Critical", warningEvents: "Warning", normalEvents: "Normal" };

describe("event severity", () => {
  it("critical 是红色（此前从文字猜颜色，Critical 落成灰色）", () => {
    expect(eventSeverityBadge("critical")).toBe("error");
    expect(eventSeverityLabel("critical", t)).toBe("Critical");
  });
  it("warning / info 各自的颜色与文案", () => {
    expect(eventSeverityBadge("warning")).toBe("warning");
    expect(eventSeverityLabel("info", t)).toBe("Normal");
  });
  it("筛选：critical 包含 error，其余按值匹配", () => {
    expect(matchesSeverityFilter("error", "critical")).toBe(true);
    expect(matchesSeverityFilter("warning", "critical")).toBe(false);
    expect(matchesSeverityFilter("warning", "warning")).toBe(true);
    expect(matchesSeverityFilter("info", "")).toBe(true);
  });
});
