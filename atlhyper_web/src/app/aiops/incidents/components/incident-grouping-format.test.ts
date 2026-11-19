import { describe, expect, it } from "vitest";
import type { Incident } from "@/api/aiops";
import { jaIncidentGrouping } from "@/i18n/locales/incident-ja";
import { zhIncidentGrouping } from "@/i18n/locales/incident-zh";
import type { IncidentLinkDetail, IncidentPathStep } from "@/types/incident-grouping";
import {
  edgeLabel,
  incidentListRows,
  isPathCollapsible,
  linkSubjectSource,
  pathSegments,
  pathText,
  rootTagLabel,
  subjectShortName,
  withDetailGroupingDefaults,
  withListGroupingDefaults,
} from "./incident-grouping-format";

const TYPE_LABELS: Record<string, string> = { pod: "Pod", service: "Service", statefulset: "StatefulSet", node: "Node" };

function incident(overrides: Partial<Incident>): Incident {
  return {
    id: "inc-1", clusterId: "c1", state: "warning", severity: "medium",
    subject: "shop/service/api", subjectType: "service", subjectNamespace: "shop", subjectName: "api",
    rootCause: "shop/service/api", peakRisk: 50, peakRiskLevel: "medium", startedAt: "2026-11-15T07:00:00Z", resolvedAt: null,
    escalatedAt: null, closeReason: "", durationS: 60, recurrence: 0, createdAt: "2026-11-15T07:00:00Z",
    rootIncidentId: "", root: null, linkedCount: 0, impacted: [],
    ...overrides,
  };
}

const SHORT_PATH: IncidentPathStep[] = [
  { entityKey: "shop/service/db", edge: "" },
  { entityKey: "shop/pod/db-0", edge: "selects" },
];
const LONG_PATH: IncidentPathStep[] = [
  { entityKey: "shop/service/payment", edge: "" },
  { entityKey: "shop/service/orders", edge: "calls" },
  { entityKey: "shop/service/db", edge: "calls" },
  { entityKey: "shop/pod/db-0", edge: "selects" },
];

describe("withListGroupingDefaults", () => {
  it("老版本后端没有三项时按「不是受影响事件、没有关联」补齐", () => {
    const { rootIncidentId: _root, linkedCount: _count, impacted: _impacted, ...legacy } = incident({});
    expect(withListGroupingDefaults<Partial<Incident>>(legacy)).toMatchObject({
      rootIncidentId: "", root: null, linkedCount: 0, impacted: [],
    });
  });

  it("后端给了就原样保留", () => {
    const summary = { incidentId: "inc-2", subject: "shop/service/db", subjectType: "service", state: "warning", severity: "low" };
    const filled = withListGroupingDefaults(incident({ linkedCount: 1, impacted: [summary] }));
    expect(filled.linkedCount).toBe(1);
    expect(filled.impacted).toEqual([summary]);
  });
});

describe("withDetailGroupingDefaults", () => {
  it("没有 root / impacted 时补成 null 与空数组", () => {
    expect(withDetailGroupingDefaults({})).toEqual({ root: null, impacted: [] });
  });

  it("关联缺路径或归入时刻时补空，其余保留", () => {
    const partial = { incidentId: "inc-7", subject: "shop/statefulset/db", subjectType: "statefulset", state: "incident", severity: "critical" };
    const filled = withDetailGroupingDefaults({ root: partial as IncidentLinkDetail, impacted: [partial as IncidentLinkDetail] });
    expect(filled.root).toEqual({ ...partial, path: [], linkedAt: "" });
    expect(filled.impacted[0]).toEqual({ ...partial, path: [], linkedAt: "" });
  });
});

describe("linkSubjectSource / subjectShortName", () => {
  it("从主体键拆出命名空间与名称，类型用后端给的", () => {
    const source = linkSubjectSource({ incidentId: "inc-2", subject: "shop/service/db", subjectType: "service", state: "", severity: "" });
    expect(source).toMatchObject({ subjectNamespace: "shop", subjectName: "db", subjectType: "service" });
  });

  it("集群级对象没有命名空间，只写名称", () => {
    const source = linkSubjectSource({ incidentId: "inc-3", subject: "_cluster/node/worker-1", subjectType: "node", state: "", severity: "" });
    expect(subjectShortName({ namespace: source.subjectNamespace, name: source.subjectName })).toBe("worker-1");
    expect(subjectShortName({ namespace: "shop", name: "db" })).toBe("shop/db");
  });
});

describe("incidentListRows", () => {
  it("平铺：受影响事件配源头小标签；源头也在本页时带主体，否则只有事件号", () => {
    const rows = incidentListRows([
      incident({ id: "inc-7", subject: "shop/statefulset/db", subjectType: "statefulset", subjectName: "db", linkedCount: 2 }),
      incident({ id: "inc-8", rootIncidentId: "inc-7" }),
      incident({ id: "inc-9", rootIncidentId: "inc-99" }),
    ]);
    expect(rows[0].rootTag).toBeNull();
    expect(rows[1].rootTag).toEqual({
      incidentId: "inc-7",
      subject: { key: "shop/statefulset/db", namespace: "shop", type: "statefulset", name: "db" },
    });
    expect(rows[2].rootTag).toEqual({ incidentId: "inc-99", subject: null });
  });

  it("不改变顺序与条数；缺字段的行补齐", () => {
    const { impacted: _impacted, ...legacy } = incident({ id: "inc-1" });
    const rows = incidentListRows([legacy as Incident, incident({ id: "inc-2" })]);
    expect(rows.map((row) => row.incident.id)).toEqual(["inc-1", "inc-2"]);
    expect(rows[0].incident.impacted).toEqual([]);
  });
});

describe("rootTagLabel", () => {
  it("知道主体：类型 + 命名空间 / 名称", () => {
    const tag = { incidentId: "inc-7", subject: { key: "shop/statefulset/db", namespace: "shop", type: "statefulset", name: "db" } };
    expect(rootTagLabel(tag, zhIncidentGrouping, TYPE_LABELS)).toBe("源头：StatefulSet shop/db");
    expect(rootTagLabel(tag, jaIncidentGrouping, TYPE_LABELS)).toBe("発生源：StatefulSet shop/db");
  });

  it("不知道主体：退回源头事件号", () => {
    expect(rootTagLabel({ incidentId: "inc-99", subject: null }, zhIncidentGrouping, TYPE_LABELS)).toBe("源头：inc-99");
  });
});

describe("edgeLabel", () => {
  it("四种边按 i18n 显示；起点没有边", () => {
    expect(["calls", "selects", "runs_on", "routes_to"].map((edge) => edgeLabel(edge, zhIncidentGrouping))).toEqual([
      "调用", "选中", "运行在", "路由到",
    ]);
    expect(edgeLabel("", zhIncidentGrouping)).toBe("");
  });

  it("中日文案齐全，且没见过的边类型原样显示", () => {
    for (const edge of ["calls", "selects", "runs_on", "routes_to"]) {
      expect(edgeLabel(edge, jaIncidentGrouping)).not.toBe(edge);
    }
    expect(edgeLabel("mounts", zhIncidentGrouping)).toBe("mounts");
  });
});

describe("pathSegments / pathText", () => {
  it("实体名（类型）与边交替，起点前没有边", () => {
    const segments = pathSegments(SHORT_PATH, zhIncidentGrouping, TYPE_LABELS, true);
    expect(segments).toEqual([
      { kind: "entity", entityKey: "shop/service/db", name: "db", typeLabel: "Service" },
      { kind: "edge", edge: "selects", label: "选中" },
      { kind: "entity", entityKey: "shop/pod/db-0", name: "db-0", typeLabel: "Pod" },
    ]);
    expect(pathText(segments, zhIncidentGrouping)).toBe("db (Service) → 选中 → db-0 (Pod)");
  });

  it("实体多于 3 个才可折叠；折叠后只留起点、进入终点的边与终点", () => {
    expect(isPathCollapsible(SHORT_PATH)).toBe(false);
    expect(isPathCollapsible(LONG_PATH.slice(0, 3))).toBe(false);
    expect(isPathCollapsible(LONG_PATH)).toBe(true);
    const collapsed = pathSegments(LONG_PATH, zhIncidentGrouping, TYPE_LABELS, true);
    expect(pathText(collapsed, zhIncidentGrouping)).toBe("payment (Service) → （省略 2 个） → 选中 → db-0 (Pod)");
  });

  it("展开时给出完整路径", () => {
    const expanded = pathSegments(LONG_PATH, jaIncidentGrouping, TYPE_LABELS, false);
    expect(pathText(expanded, jaIncidentGrouping)).toBe(
      "payment (Service) → 呼び出し → orders (Service) → 呼び出し → db (Service) → セレクト → db-0 (Pod)",
    );
  });

  it("不可折叠的路径不受 collapsed 影响；没见过的实体类型原样显示", () => {
    expect(pathSegments(SHORT_PATH, zhIncidentGrouping, TYPE_LABELS, true)).toEqual(
      pathSegments(SHORT_PATH, zhIncidentGrouping, TYPE_LABELS, false),
    );
    const [first] = pathSegments([{ entityKey: "shop/gateway/edge", edge: "" }], zhIncidentGrouping, TYPE_LABELS, true);
    expect(first).toMatchObject({ kind: "entity", typeLabel: "gateway" });
  });

  it("空路径没有段", () => {
    expect(pathSegments([], zhIncidentGrouping, TYPE_LABELS, true)).toEqual([]);
  });
});

describe("incidentListRows：源头概要来自后端", () => {
  it("受影响事件带了 root 概要时，源头小标签用它，不依赖源头是否在本页", () => {
    const rows = incidentListRows([
      incident({
        id: "inc-svc", rootIncidentId: "inc-db",
        root: {
          incidentId: "inc-db", subject: "shop/statefulset/db", subjectType: "statefulset",
          subjectNamespace: "shop", subjectName: "db", state: "incident", severity: "critical",
        },
      }),
    ]);
    expect(rows[0].rootTag?.incidentId).toBe("inc-db");
    expect(rows[0].rootTag?.subject?.name).toBe("db");
    expect(rows[0].rootTag?.subject?.type).toBe("statefulset");
  });
});
