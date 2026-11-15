/**
 * 拓扑归并的展示映射（纯函数）：列表的源头 / 受影响标记、依赖路径的显示、老版本后端的缺省补齐
 *
 * 契约：docs/design/active/aiops-topology-grouping-design.md「接口（契约）」。
 * 分组由后端完成（grouped=true 时受影响事件折叠进源头的 impacted），这里不做分组与聚合，
 * 只做「码 → 文案」映射、缺省补齐与路径的折叠显示。
 */
import type { Incident } from "@/api/aiops";
import { incidentSubject, parseEntityKey, type IncidentSubject, type IncidentSubjectSource } from "@/lib/entity-key";
import type { IncidentGroupingTranslations } from "@/types/i18n-incident";
import type {
  IncidentDetailGrouping,
  IncidentLinkDetail,
  IncidentLinkSummary,
  IncidentListGrouping,
  IncidentPathStep,
} from "@/types/incident-grouping";
import { fillTemplate } from "./incident-view-format";

/**
 * 补齐列表新增的三项：老版本后端没有这些字段时，按「不是受影响事件、没有关联」显示。
 * 老版本后端也不认 grouped 参数，返回的就是平铺列表，补齐后照常显示。
 */
export function withListGroupingDefaults<T extends Partial<IncidentListGrouping>>(incident: T): T & IncidentListGrouping {
  return {
    ...incident,
    rootIncidentId: incident.rootIncidentId ?? "",
    linkedCount: incident.linkedCount ?? 0,
    impacted: incident.impacted ?? [],
  };
}

/** 补齐详情新增的两项（同 withDetailViewDefaults 的做法）；路径、归入时刻缺失时按空 */
export function withDetailGroupingDefaults<T extends Partial<IncidentDetailGrouping>>(detail: T): T & IncidentDetailGrouping {
  const withPath = (link: IncidentLinkDetail): IncidentLinkDetail => ({
    ...link,
    path: link.path ?? [],
    linkedAt: link.linkedAt ?? "",
  });
  return {
    ...detail,
    root: detail.root ? withPath(detail.root) : null,
    impacted: (detail.impacted ?? []).map(withPath),
  };
}

/** 关联另一端的事件只给了主体键与类型：拆出命名空间与名称，拼成显示主体用的形状 */
export function linkSubjectSource(link: IncidentLinkSummary): IncidentSubjectSource {
  const { namespace, name } = parseEntityKey(link.subject);
  return {
    subject: link.subject,
    subjectType: link.subjectType,
    subjectNamespace: namespace,
    subjectName: name,
    rootCause: "",
  };
}

/** 列表一行的显示数据 */
export interface IncidentListRowView {
  incident: Incident;
  /**
   * 平铺时受影响事件的「源头」小标签：incidentId 用来打开源头详情；
   * subject 是源头的主体——列表接口只给了源头事件号，源头恰好也在这一页里时才知道主体，否则为 null（只显示事件号）
   */
  rootTag: { incidentId: string; subject: IncidentSubject | null } | null;
}

/**
 * 列表的行：补齐缺省字段，给受影响事件配上源头小标签。
 *
 * grouped=true 时后端已把受影响事件折叠进源头，这里不会出现 rootIncidentId 非空的行；
 * 平铺时才有。源头的主体只在本页响应里查，查不到就只显示事件号（契约问题见汇报）。
 */
export function incidentListRows(incidents: Incident[]): IncidentListRowView[] {
  const normalized = incidents.map(withListGroupingDefaults);
  const byId = new Map(normalized.map((incident) => [incident.id, incident]));
  return normalized.map((incident) => {
    if (incident.rootIncidentId === "") return { incident, rootTag: null };
    const root = byId.get(incident.rootIncidentId);
    return {
      incident,
      rootTag: { incidentId: incident.rootIncidentId, subject: root ? incidentSubject(root) : null },
    };
  });
}

/** 主体的短写法：命名空间 / 名称；集群级对象只有名称 */
export function subjectShortName(subject: Pick<IncidentSubject, "namespace" | "name">): string {
  return subject.namespace === "" ? subject.name : `${subject.namespace}/${subject.name}`;
}

/** 平铺时受影响事件的「源头：<主体>」：主体 = 类型 + 命名空间 / 名称；不知道主体时用源头事件号 */
export function rootTagLabel(
  rootTag: NonNullable<IncidentListRowView["rootTag"]>,
  texts: IncidentGroupingTranslations,
  typeLabels: Record<string, string>,
): string {
  const { subject } = rootTag;
  const label = subject ? `${typeLabels[subject.type] ?? subject.type} ${subjectShortName(subject)}` : rootTag.incidentId;
  return fillTemplate(texts.rootTag, { subject: label });
}

/** 边类型 → 文案；起点（空串）没有边；没见过的取值原样显示 */
export function edgeLabel(edge: string, texts: IncidentGroupingTranslations): string {
  if (edge === "") return "";
  const labels: Record<string, string> = texts.edge;
  return labels[edge] ?? edge;
}

/** 依赖路径的一段：实体、边，或折叠起来的中间部分 */
export type PathSegment =
  | { kind: "entity"; entityKey: string; name: string; typeLabel: string }
  | { kind: "edge"; edge: string; label: string }
  | { kind: "gap"; hidden: number };

/** 路径的实体多于这个数时可以折叠：只留起点与终点 */
export const PATH_COLLAPSE_THRESHOLD = 3;

export function isPathCollapsible(path: IncidentPathStep[]): boolean {
  return path.length > PATH_COLLAPSE_THRESHOLD;
}

/**
 * 路径 → 显示的段：实体名 +（类型），相邻实体之间是边的文案。
 * collapsed 且可折叠时，中间的实体收成一段 gap，只留起点、终点与进入终点的那条边。
 */
export function pathSegments(
  path: IncidentPathStep[],
  texts: IncidentGroupingTranslations,
  typeLabels: Record<string, string>,
  collapsed: boolean,
): PathSegment[] {
  const entityOf = (step: IncidentPathStep): PathSegment => {
    const { type, name } = parseEntityKey(step.entityKey);
    return { kind: "entity", entityKey: step.entityKey, name, typeLabel: typeLabels[type] ?? type };
  };
  const edgeOf = (step: IncidentPathStep): PathSegment[] =>
    step.edge === "" ? [] : [{ kind: "edge", edge: step.edge, label: edgeLabel(step.edge, texts) }];

  if (collapsed && isPathCollapsible(path)) {
    const last = path[path.length - 1];
    return [entityOf(path[0]), { kind: "gap", hidden: path.length - 2 }, ...edgeOf(last), entityOf(last)];
  }
  return path.flatMap((step, index) => (index === 0 ? [entityOf(step)] : [...edgeOf(step), entityOf(step)]));
}

/** 路径的纯文本写法（悬停提示用）：api (Service) → 调用 → db (Service) */
export function pathText(segments: PathSegment[], texts: IncidentGroupingTranslations): string {
  return segments
    .map((segment) => {
      if (segment.kind === "entity") return `${segment.name} (${segment.typeLabel})`;
      if (segment.kind === "edge") return segment.label;
      return gapText(segment.hidden, texts);
    })
    .join(" → ");
}

/** 折叠段的文案 */
export function gapText(hidden: number, texts: IncidentGroupingTranslations): string {
  return fillTemplate(texts.pathGap, { count: hidden });
}
