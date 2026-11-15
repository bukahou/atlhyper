import { describe, expect, it } from "vitest";
import type { IncidentSubject } from "@/lib/entity-key";
import type { IncidentMember } from "@/types/incident-view";
import { buildNextSteps, logsTargetPod } from "./incident-next-steps";

function pod(name: string, overrides: Partial<IncidentMember> = {}): IncidentMember {
  return {
    entityKey: `shop/pod/${name}`,
    entityType: "pod",
    namespace: "shop",
    name,
    role: "affected",
    rFinal: 50,
    isLead: false,
    present: true,
    status: "Running",
    restarts: 0,
    ready: "1/1",
    ...overrides,
  };
}

const DEPLOYMENT: IncidentSubject = { key: "shop/deployment/api", type: "deployment", namespace: "shop", name: "api" };

describe("buildNextSteps", () => {
  it("工作负载：主体详情 → Pod 列表（按命名空间 + 名称）→ 相关事件 → 日志 → AI", () => {
    const members = [pod("api-1", { isLead: true }), pod("api-2"), pod("api-3")];
    expect(buildNextSteps(DEPLOYMENT, members)).toEqual([
      { kind: "subject", target: { type: "deployment", name: "api", namespace: "shop" } },
      { kind: "pods", href: "/cluster/pod?namespace=shop&search=api", count: 3 },
      { kind: "events", href: "/cluster/event?namespace=shop&search=api" },
      { kind: "logs", namespace: "shop", podName: "api-1" },
      { kind: "ai" },
    ]);
  });

  it("节点：看该节点上的 Pod；事件不带命名空间；成员不是 Pod，没有日志", () => {
    const node: IncidentSubject = { key: "_cluster/node/worker-1", type: "node", namespace: "", name: "worker-1" };
    const members: IncidentMember[] = [
      { ...pod("worker-1"), entityKey: "_cluster/node/worker-1", entityType: "node", namespace: "", isLead: true },
    ];
    expect(buildNextSteps(node, members).map((step) => step.kind)).toEqual(["subject", "nodePods", "events", "ai"]);
    expect(buildNextSteps(node, members)[1]).toEqual({ kind: "nodePods", href: "/cluster/pod?node=worker-1" });
    expect(buildNextSteps(node, members)[2]).toEqual({ kind: "events", href: "/cluster/event?search=worker-1" });
  });

  it("裸 Pod / Service：没有 Pod 列表这一步", () => {
    const bare: IncidentSubject = { key: "shop/pod/job-x", type: "pod", namespace: "shop", name: "job-x" };
    expect(buildNextSteps(bare, [pod("job-x", { isLead: true })]).map((step) => step.kind)).toEqual([
      "subject", "events", "logs", "ai",
    ]);
  });

  it("认不出的主体类型：不给详情入口，其余照常", () => {
    const unknown: IncidentSubject = { key: "x", type: "unknown", namespace: "", name: "x" };
    expect(buildNextSteps(unknown, []).map((step) => step.kind)).toEqual(["events", "ai"]);
  });

  it("成员全都不在了：没有日志这一步", () => {
    const members = [pod("api-1", { isLead: true, present: false }), pod("api-2", { present: false })];
    expect(buildNextSteps(DEPLOYMENT, members).some((step) => step.kind === "logs")).toBe(false);
  });
});

describe("logsTargetPod", () => {
  it("风险最高的优先", () => {
    expect(logsTargetPod([pod("a"), pod("b", { isLead: true })])?.name).toBe("b");
  });

  it("风险最高的已不存在：取第一个还在的 Pod", () => {
    expect(logsTargetPod([pod("a", { isLead: true, present: false }), pod("b", { present: false }), pod("c")])?.name).toBe("c");
  });

  it("没有 Pod 成员", () => {
    expect(logsTargetPod([])).toBeNull();
  });
});
