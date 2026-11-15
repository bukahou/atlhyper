import { describe, expect, it } from "vitest";
import type { NotifyRule } from "@/types/notify";
import { movedRuleIds, ruleBodyFromDraft, ruleDraftFrom, ruleDraftProblems } from "./notify-rule-input";

const savedRule: NotifyRule = {
  id: 1,
  name: "生产高严重度",
  position: 1,
  enabled: true,
  clusterIds: ["prod"],
  sources: ["incident"],
  minSeverity: "high",
  receiverIds: [1, 2],
  groupWaitSeconds: null,
  groupIntervalSeconds: null,
  repeatIntervalSeconds: 3600,
  createdAt: "2026-11-01T00:00:00Z",
  updatedAt: "2026-11-01T00:00:00Z",
};

describe("ruleDraftFrom", () => {
  it("新建：启用、不限集群与来源、最低严重度 low、三个时间参数都不覆盖", () => {
    expect(ruleDraftFrom()).toEqual({
      name: "",
      enabled: true,
      clusterIds: [],
      sources: [],
      minSeverity: "low",
      receiverIds: [],
      groupWaitSeconds: "",
      groupIntervalSeconds: "",
      repeatIntervalSeconds: "",
    });
  });

  it("编辑：null 的时间参数显示为空，有值的显示为数字", () => {
    expect(ruleDraftFrom(savedRule)).toMatchObject({
      name: "生产高严重度",
      clusterIds: ["prod"],
      sources: ["incident"],
      minSeverity: "high",
      receiverIds: [1, 2],
      groupWaitSeconds: "",
      groupIntervalSeconds: "",
      repeatIntervalSeconds: "3600",
    });
  });

  it("后端把空数组序列化成 null 时按空数组处理", () => {
    const nullArrays = { ...savedRule, clusterIds: null, sources: null, receiverIds: null } as unknown as NotifyRule;
    expect(ruleDraftFrom(nullArrays)).toMatchObject({ clusterIds: [], sources: [], receiverIds: [] });
  });
});

describe("ruleDraftProblems", () => {
  it("已保存的规则原样填回表单是合规的", () => {
    expect(ruleDraftProblems(ruleDraftFrom(savedRule))).toEqual([]);
  });

  it("名称必填；只有空白也算没填", () => {
    expect(ruleDraftProblems({ ...ruleDraftFrom(savedRule), name: "  " })).toEqual(["nameRequired"]);
  });

  it("时间参数留空合法（用全局值）", () => {
    expect(ruleDraftProblems({ ...ruleDraftFrom(savedRule), repeatIntervalSeconds: "" })).toEqual([]);
  });

  it("时间参数填了就按策略里同名项的范围检查", () => {
    expect(
      ruleDraftProblems({
        ...ruleDraftFrom(savedRule),
        groupWaitSeconds: "601",
        groupIntervalSeconds: "29",
        repeatIntervalSeconds: "abc",
      }),
    ).toEqual(["groupWaitSeconds", "groupIntervalSeconds", "repeatIntervalSeconds"]);
  });
});

describe("ruleBodyFromDraft", () => {
  it("空的时间参数发 null，填了的发数字；名称去掉首尾空白；不带 id 与 position", () => {
    expect(ruleBodyFromDraft({ ...ruleDraftFrom(savedRule), name: " 生产高严重度 ", groupWaitSeconds: "10" })).toEqual({
      name: "生产高严重度",
      enabled: true,
      clusterIds: ["prod"],
      sources: ["incident"],
      minSeverity: "high",
      receiverIds: [1, 2],
      groupWaitSeconds: 10,
      groupIntervalSeconds: null,
      repeatIntervalSeconds: 3600,
    });
  });
});

describe("movedRuleIds", () => {
  it("上移：与前一条对调，返回全部规则的完整排列", () => {
    expect(movedRuleIds([3, 1, 2], 1, "up")).toEqual([1, 3, 2]);
  });

  it("下移：与后一条对调", () => {
    expect(movedRuleIds([3, 1, 2], 1, "down")).toEqual([3, 2, 1]);
  });

  it("第一条不能再上移，最后一条不能再下移", () => {
    expect(movedRuleIds([3, 1, 2], 3, "up")).toBeNull();
    expect(movedRuleIds([3, 1, 2], 2, "down")).toBeNull();
  });

  it("id 不在列表里：没有可做的移动", () => {
    expect(movedRuleIds([3, 1, 2], 9, "up")).toBeNull();
    expect(movedRuleIds([], 1, "down")).toBeNull();
  });

  it("不改传入的数组", () => {
    const current = [3, 1, 2];
    movedRuleIds(current, 1, "up");
    expect(current).toEqual([3, 1, 2]);
  });
});
