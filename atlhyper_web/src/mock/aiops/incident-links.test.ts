import { describe, expect, it } from "vitest";
import { INCIDENT_SEEDS, mockIncidents } from "./incidents";
import { mockIncidentDetail } from "./incident-detail";

/** 演示数据要守住拓扑归并契约的约定，界面才能拿它当真实响应开发 */
describe("拓扑归并的演示数据与契约一致", () => {
  const flat = mockIncidents({ cluster: "demo" });
  const grouped = mockIncidents({ cluster: "demo", grouped: true });

  it("平铺：每条都带三项；受影响事件单独出现、带源头事件号，impacted 一律为空", () => {
    expect(flat).toHaveLength(INCIDENT_SEEDS.length);
    for (const incident of flat) {
      expect(typeof incident.rootIncidentId).toBe("string");
      expect(typeof incident.linkedCount).toBe("number");
      expect(incident.impacted).toEqual([]);
    }
    const impactedRows = flat.filter((incident) => incident.rootIncidentId !== "");
    expect(impactedRows.map((incident) => incident.id).sort()).toEqual(["inc-008", "inc-009"]);
    expect(flat.find((incident) => incident.id === "inc-007")?.linkedCount).toBe(2);
  });

  it("grouped=true：受影响事件不单独出现，折叠在源头的 impacted 里；impacted 条数等于 linkedCount", () => {
    expect(grouped.some((incident) => incident.rootIncidentId !== "")).toBe(false);
    expect(grouped).toHaveLength(flat.length - 2);
    for (const incident of grouped) {
      if (incident.linkedCount === 0) expect(incident.impacted).toEqual([]);
      else expect(incident.impacted).toHaveLength(incident.linkedCount);
    }
    const root = grouped.find((incident) => incident.id === "inc-007");
    expect(root?.impacted.map((item) => item.incidentId)).toEqual(["inc-008", "inc-009"]);
    expect(root?.impacted[0]).toEqual({
      incidentId: "inc-008", subject: "shop/service/db", subjectType: "service", state: "warning", severity: "medium",
    });
  });

  it("详情：受影响事件的 root 与源头详情的 impacted 互相对得上", () => {
    const rootDetail = mockIncidentDetail("inc-007");
    expect(rootDetail.root).toBeNull();
    expect(rootDetail.impacted.map((item) => item.incidentId)).toEqual(["inc-008", "inc-009"]);
    for (const item of rootDetail.impacted) {
      const child = mockIncidentDetail(item.incidentId);
      expect(child.root?.incidentId).toBe("inc-007");
      expect(child.root?.path).toEqual(item.path);
      expect(child.impacted).toEqual([]);
    }
  });

  it("路径：第一步的 edge 为空串、从受影响事件的成员出发，最后一步落在源头的成员上", () => {
    const rootDetail = mockIncidentDetail("inc-007");
    const rootMembers = new Set(rootDetail.members.map((member) => member.entityKey));
    for (const item of rootDetail.impacted) {
      const child = mockIncidentDetail(item.incidentId);
      const childMembers = new Set(child.members.map((member) => member.entityKey));
      expect(item.path[0].edge).toBe("");
      expect(childMembers.has(item.path[0].entityKey)).toBe(true);
      expect(rootMembers.has(item.path[item.path.length - 1].entityKey)).toBe(true);
      for (const step of item.path.slice(1)) {
        expect(["calls", "selects", "runs_on", "routes_to"]).toContain(step.edge);
      }
    }
  });

  it("不在任何组里的事件：root 为 null、impacted 为空", () => {
    const detail = mockIncidentDetail("inc-001");
    expect(detail.root).toBeNull();
    expect(detail.impacted).toEqual([]);
  });
});
