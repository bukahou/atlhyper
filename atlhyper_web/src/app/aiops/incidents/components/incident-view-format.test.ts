import { describe, expect, it } from "vitest";
import { jaIncidentDetail } from "@/i18n/locales/incident-ja";
import { zhIncidentDetail } from "@/i18n/locales/incident-zh";
import type { IncidentDetailView, IncidentHeadline, IncidentImpact, IncidentSignal } from "@/types/incident-view";
import {
  evidenceTone,
  fillTemplate,
  headlineSentence,
  impactItems,
  memberStatusTone,
  signalReasonText,
  verdictMetaParts,
  withDetailViewDefaults,
} from "./incident-view-format";

const NO_IMPACT: IncidentImpact = { desiredReplicas: null, availableReplicas: null, restartsTotal: null };
const ALL_SIGNALS: IncidentSignal[] = [
  "container_crash", "workload_unavailable", "k8s_event", "restarts", "not_ready", "resource",
  "node_pressure", "apm_errors", "apm_latency", "log_errors", "other",
];

function headline(overrides: Partial<IncidentHeadline>): IncidentHeadline {
  return { signal: "container_crash", reason: "CrashLoopBackOff", affectedMembers: 3, totalMembers: 3, ...overrides };
}

describe("fillTemplate", () => {
  it("替换全部占位，数字转成字符串", () => {
    expect(fillTemplate("{a} / {b} / {a}", { a: 1, b: "x" })).toBe("1 / x / 1");
  });

  it("没提供的占位原样保留，方便发现漏填", () => {
    expect(fillTemplate("{a} {missing}", { a: "ok" })).toBe("ok {missing}");
  });
});

describe("signalReasonText", () => {
  it("工作负载不可用的取值是百分比整数，补上 %", () => {
    expect(signalReasonText("workload_unavailable", "67")).toBe("67%");
  });

  it("其余信号原样；空串仍是空串", () => {
    expect(signalReasonText("container_crash", "OOMKilled")).toBe("OOMKilled");
    expect(signalReasonText("workload_unavailable", "")).toBe("");
  });
});

describe("headlineSentence", () => {
  const base = { impact: NO_IMPACT, subjectType: "deployment", subjectTypeLabel: "Deployment", hasDetections: true };

  it("成员全部中招：N 个 Pod 全部 + 信号说法 + 原因码", () => {
    expect(headlineSentence({ ...base, headline: headline({}) }, zhIncidentDetail)).toBe(
      "3 个 Pod 全部容器异常（CrashLoopBackOff）。",
    );
  });

  it("部分成员中招", () => {
    expect(
      headlineSentence({ ...base, headline: headline({ affectedMembers: 1, reason: "" }) }, zhIncidentDetail),
    ).toBe("3 个 Pod 中有 1 个容器异常。");
  });

  it("工作负载已无可用副本时补半句", () => {
    const impact: IncidentImpact = { desiredReplicas: 3, availableReplicas: 0, restartsTotal: 18 };
    expect(headlineSentence({ ...base, impact, headline: headline({}) }, zhIncidentDetail)).toBe(
      "3 个 Pod 全部容器异常（CrashLoopBackOff），Deployment 已无可用副本。",
    );
  });

  it("主证据就是「不可用」：按主体说，不再补半句", () => {
    const impact: IncidentImpact = { desiredReplicas: 3, availableReplicas: 0, restartsTotal: 18 };
    const sentence = headlineSentence(
      { ...base, impact, headline: headline({ signal: "workload_unavailable", reason: "100" }) },
      zhIncidentDetail,
    );
    expect(sentence).toBe("Deployment 不可用 100%。");
  });

  it("不可用但没有百分比", () => {
    expect(
      headlineSentence({ ...base, headline: headline({ signal: "workload_unavailable", reason: "" }) }, zhIncidentDetail),
    ).toBe("Deployment 不可用。");
  });

  it("只有一个成员的主体：按主体类型说", () => {
    expect(
      headlineSentence(
        {
          impact: NO_IMPACT,
          subjectType: "node",
          subjectTypeLabel: "Node",
          hasDetections: true,
          headline: headline({ signal: "node_pressure", reason: "MemoryPressure", affectedMembers: 1, totalMembers: 1 }),
        },
        zhIncidentDetail,
      ),
    ).toBe("Node：出现节点压力（MemoryPressure）。");
  });

  it("不是工作负载的多成员主体叫「成员」", () => {
    expect(
      headlineSentence({ ...base, subjectType: "service", subjectTypeLabel: "Service", headline: headline({}) }, zhIncidentDetail),
    ).toBe("3 个成员全部容器异常（CrashLoopBackOff）。");
  });

  it("headline 为 null、也没有检测记录：说明没有本地异常信号（依赖传播）", () => {
    expect(headlineSentence({ ...base, hasDetections: false, headline: null }, zhIncidentDetail)).toBe(
      zhIncidentDetail.headline.none,
    );
  });

  it("headline 为 null、但有检测记录：是改版前的事件，不说成依赖传播", () => {
    expect(headlineSentence({ ...base, headline: null }, zhIncidentDetail)).toBe(zhIncidentDetail.headline.legacy);
  });

  it("日文模板同样填满，没有遗留占位", () => {
    expect(headlineSentence({ ...base, headline: headline({ affectedMembers: 2 }) }, jaIncidentDetail)).toBe(
      "3 個の Pod のうち 2 個でコンテナ異常（CrashLoopBackOff）。",
    );
  });

  it("每种信号在中日文里都有说法，句子里不留 {占位}", () => {
    for (const texts of [zhIncidentDetail, jaIncidentDetail]) {
      for (const signal of ALL_SIGNALS) {
        for (const totals of [{ affectedMembers: 1, totalMembers: 1 }, { affectedMembers: 1, totalMembers: 3 }]) {
          const sentence = headlineSentence({ ...base, headline: headline({ signal, ...totals }) }, texts);
          expect(sentence).not.toMatch(/\{\w+\}/);
          expect(texts.signalName[signal]).toBeTruthy();
        }
      }
    }
  });
});

describe("impactItems", () => {
  it("副本、重启、峰值风险都有", () => {
    expect(
      impactItems({ desiredReplicas: 3, availableReplicas: 0, restartsTotal: 18 }, 89.6, zhIncidentDetail),
    ).toEqual(["可用副本 0 / 3", "容器重启共 18 次", "峰值风险 90（满分 100）"]);
  });

  it("为 null 的项不显示；副本数缺一项也不显示", () => {
    expect(impactItems({ desiredReplicas: 3, availableReplicas: null, restartsTotal: null }, 40, zhIncidentDetail)).toEqual([
      "峰值风险 40（满分 100）",
    ]);
  });

  it("重启为 0 仍显示（0 不是 null）", () => {
    expect(impactItems({ ...NO_IMPACT, restartsTotal: 0 }, 10, zhIncidentDetail)).toContain("容器重启共 0 次");
  });
});

describe("verdictMetaParts", () => {
  const startedAt = new Date(2026, 10, 15, 14, 53).toISOString();
  const escalatedAt = new Date(2026, 10, 15, 14, 58).toISOString();

  it("进行中：已持续 · 开始 · 升级 · 未复发", () => {
    expect(
      verdictMetaParts({ state: "incident", durationS: 12 * 60, startedAt, escalatedAt, recurrence: 0 }, zhIncidentDetail, "分钟"),
    ).toEqual(["已持续 12 分钟", "11/15 14:53 开始", "11/15 14:58 升级", "未复发"]);
  });

  it("恢复观察 / 已关闭：时长已定下，说「持续了」；未升级；复发次数", () => {
    expect(
      verdictMetaParts({ state: "recovery", durationS: 95 * 60, startedAt, escalatedAt: null, recurrence: 3 }, zhIncidentDetail, "分钟"),
    ).toEqual(["持续了 1h 35m", "11/15 14:53 开始", "未升级", "复发 3 次"]);
  });
});

describe("evidenceTone / memberStatusTone", () => {
  it("每种信号都有色条；崩溃红色、资源橙色、K8s 事件灰色", () => {
    expect(evidenceTone("container_crash")).toBe("red");
    expect(evidenceTone("resource")).toBe("amber");
    expect(evidenceTone("k8s_event")).toBe("gray");
    for (const signal of ALL_SIGNALS) expect(["red", "amber", "gray"]).toContain(evidenceTone(signal));
  });

  it("没见过的信号按灰色", () => {
    expect(evidenceTone("future_signal" as IncidentSignal)).toBe("gray");
  });

  it("K8s 状态字符串分组", () => {
    expect(memberStatusTone("Running")).toBe("ok");
    expect(memberStatusTone("Ready")).toBe("ok");
    expect(memberStatusTone("ContainerCreating")).toBe("pending");
    expect(memberStatusTone("CrashLoopBackOff")).toBe("bad");
    expect(memberStatusTone("OOMKilled")).toBe("bad");
  });
});

describe("withDetailViewDefaults", () => {
  it("老版本响应没有新字段：补成空", () => {
    const legacyResponse: { id: string } & Partial<IncidentDetailView> = { id: "inc-1" };
    expect(withDetailViewDefaults(legacyResponse)).toEqual({
      id: "inc-1",
      headline: null,
      impact: NO_IMPACT,
      evidence: [],
      members: [],
      timelineView: [],
    });
  });

  it("已有的字段原样保留", () => {
    const impact: IncidentImpact = { desiredReplicas: 1, availableReplicas: 1, restartsTotal: 2 };
    expect(withDetailViewDefaults({ impact, headline: headline({}) }).impact).toBe(impact);
  });
});
