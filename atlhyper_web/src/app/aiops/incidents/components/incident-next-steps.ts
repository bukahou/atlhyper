/**
 * 事件详情的「下一步」：按主体类型拼出去哪查（纯函数）
 *
 * 契约约定由前端决定（页面路由是前端的事）：
 * - 主体本身 → 资源详情弹窗（与实体链接同一个入口）
 * - 工作负载 → Pod 列表（按命名空间 + 名称预填筛选）；节点 → 该节点上的 Pod
 * - 相关 K8s 事件 → 事件列表（按命名空间 + 名称预填筛选）
 * - 容器日志 → 风险最高、且当前还在的那个 Pod（需要 pod.logs.read）
 * - AI 分析 → 弹窗底部现有的 AI 分析区（需要 ai.analyze）
 */
import { eventListLink, podListLink } from "@/lib/signal-link";
import type { IncidentSubject } from "@/lib/entity-key";
import type { EntityDetailTarget, EntityType } from "@/types/entity-detail";
import type { IncidentMember } from "@/types/incident-view";

export type IncidentNextStep =
  | { kind: "subject"; target: EntityDetailTarget }
  | { kind: "pods"; href: string; count: number }
  | { kind: "nodePods"; href: string }
  | { kind: "events"; href: string }
  | { kind: "logs"; namespace: string; podName: string }
  | { kind: "ai" };

/** 能打开资源详情弹窗的主体类型（EntityDetailRenderer 认识的） */
const DETAIL_TYPES = new Set<string>([
  "pod", "node", "service", "ingress", "deployment", "statefulset", "daemonset", "job", "cronjob",
]);

/** 成员是 Pod、Pod 名以主体名开头的主体类型 */
const WORKLOAD_TYPES = new Set(["deployment", "statefulset", "daemonset", "job", "cronjob"]);

/** 看日志的 Pod：风险最高的优先，它不在了就取第一个还在的 Pod 成员 */
export function logsTargetPod(members: IncidentMember[]): IncidentMember | null {
  const presentPods = members.filter((member) => member.entityType === "pod" && member.present);
  return presentPods.find((member) => member.isLead) ?? presentPods[0] ?? null;
}

export function buildNextSteps(subject: IncidentSubject, members: IncidentMember[]): IncidentNextStep[] {
  const steps: IncidentNextStep[] = [];

  if (DETAIL_TYPES.has(subject.type)) {
    steps.push({
      kind: "subject",
      target: { type: subject.type as EntityType, name: subject.name, namespace: subject.namespace },
    });
  }

  if (WORKLOAD_TYPES.has(subject.type)) {
    steps.push({
      kind: "pods",
      href: podListLink({ namespace: subject.namespace, search: subject.name }),
      count: members.filter((member) => member.entityType === "pod").length,
    });
  } else if (subject.type === "node") {
    steps.push({ kind: "nodePods", href: podListLink({ node: subject.name }) });
  }

  steps.push({ kind: "events", href: eventListLink({ namespace: subject.namespace, search: subject.name }) });

  const logsPod = logsTargetPod(members);
  if (logsPod) steps.push({ kind: "logs", namespace: logsPod.namespace, podName: logsPod.name });

  steps.push({ kind: "ai" });
  return steps;
}
