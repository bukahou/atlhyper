import { describe, expect, it } from "vitest";
import type { NotifyPolicy } from "@/types/notify";
import {
  isIntegerInRange,
  NOTIFY_POLICY_RANGES,
  parseInteger,
  policyNumberDraft,
  policyNumberProblems,
  policyWithNumbers,
} from "./notify-policy-input";

const policy: NotifyPolicy = {
  incidentAlertsEnabled: true,
  minSeverity: "low",
  notifyResolved: true,
  warningFlapThreshold: 3,
  groupWaitSeconds: 30,
  groupIntervalSeconds: 300,
  repeatIntervalSeconds: 14400,
  k8sEventAlertsEnabled: false,
  defaultReceiverIds: [],
};

describe("parseInteger", () => {
  it.each([
    ["0", 0],
    ["30", 30],
    [" 300 ", 300],
    ["-1", -1],
  ])("把 %j 读成 %i", (text, expected) => {
    expect(parseInteger(text)).toBe(expected);
  });

  it.each(["", " ", "1.5", "1e3", "abc", "12a", "0x10", "９"])("%j 不是十进制整数", (text) => {
    expect(parseInteger(text)).toBeNull();
  });
});

describe("isIntegerInRange", () => {
  it("两端都算在范围内", () => {
    expect(isIntegerInRange("30", { min: 30, max: 86400 })).toBe(true);
    expect(isIntegerInRange("86400", { min: 30, max: 86400 })).toBe(true);
  });

  it("范围外、不是整数都不通过", () => {
    expect(isIntegerInRange("29", { min: 30, max: 86400 })).toBe(false);
    expect(isIntegerInRange("86401", { min: 30, max: 86400 })).toBe(false);
    expect(isIntegerInRange("", { min: 0, max: 600 })).toBe(false);
  });
});

describe("NOTIFY_POLICY_RANGES", () => {
  it("与设计文档里的校验范围一致", () => {
    expect(NOTIFY_POLICY_RANGES).toEqual({
      warningFlapThreshold: { min: 0, max: 20 },
      groupWaitSeconds: { min: 0, max: 600 },
      groupIntervalSeconds: { min: 30, max: 86400 },
      repeatIntervalSeconds: { min: 300, max: 604800 },
    });
  });
});

describe("policyNumberProblems", () => {
  it("后端的默认值全部合规", () => {
    expect(policyNumberProblems(policyNumberDraft(policy))).toEqual([]);
  });

  it("0 对反复阈值与分组等待是合法的（0 = 不通知 / 不等待）", () => {
    const draft = { ...policyNumberDraft(policy), warningFlapThreshold: "0", groupWaitSeconds: "0" };
    expect(policyNumberProblems(draft)).toEqual([]);
  });

  it("逐项报出超出范围或读不出整数的项", () => {
    expect(
      policyNumberProblems({
        warningFlapThreshold: "21",
        groupWaitSeconds: "601",
        groupIntervalSeconds: "29",
        repeatIntervalSeconds: "",
      }),
    ).toEqual(["warningFlapThreshold", "groupWaitSeconds", "groupIntervalSeconds", "repeatIntervalSeconds"]);
  });

  it("只报有问题的那一项", () => {
    expect(policyNumberProblems({ ...policyNumberDraft(policy), repeatIntervalSeconds: "299" })).toEqual([
      "repeatIntervalSeconds",
    ]);
  });
});

describe("policyWithNumbers", () => {
  it("把输入框里的文本并回策略，其余项不动", () => {
    const merged = policyWithNumbers(policy, {
      warningFlapThreshold: "5",
      groupWaitSeconds: "60",
      groupIntervalSeconds: "600",
      repeatIntervalSeconds: "3600",
    });
    expect(merged).toEqual({
      ...policy,
      warningFlapThreshold: 5,
      groupWaitSeconds: 60,
      groupIntervalSeconds: 600,
      repeatIntervalSeconds: 3600,
    });
  });

  it("不改传入的策略对象", () => {
    policyWithNumbers(policy, { ...policyNumberDraft(policy), groupWaitSeconds: "60" });
    expect(policy.groupWaitSeconds).toBe(30);
  });
});
