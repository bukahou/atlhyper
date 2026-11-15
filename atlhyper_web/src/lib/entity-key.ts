/**
 * AIOps 实体键与事件主体键的拆解（纯函数）
 *
 * 键的格式由后端定：
 * - `<namespace>/<type>/<name>`：pod、service、ingress，以及工作负载主体
 *   （deployment / statefulset / daemonset / job / cronjob）
 * - `cluster/<clusterId>/node/<name>`：节点
 * 接口只给整串键（事件的 subject、rootCause，实体的 entityKey），命名空间与名称要从键里拆。
 */

export interface EntityKeyParts {
  /** 集群级对象（节点）为空串 */
  namespace: string;
  /** 小写的类型，如 pod / deployment；认不出格式时为 "unknown" */
  type: string;
  name: string;
}

export function parseEntityKey(entityKey: string): EntityKeyParts {
  const parts = entityKey.split("/");
  if (parts.length >= 4 && parts[0] === "cluster") {
    // cluster/{clusterId}/node/{name}
    return { namespace: "", type: parts[2], name: parts.slice(3).join("/") };
  }
  if (parts.length >= 3) {
    // namespace/type/name
    return { namespace: parts[0], type: parts[1], name: parts.slice(2).join("/") };
  }
  if (parts.length === 2) {
    return { namespace: "", type: parts[0], name: parts[1] };
  }
  return { namespace: "", type: "unknown", name: entityKey };
}

/** 事件里用来确定主体的三个字段 */
export interface IncidentSubjectSource {
  /** 主体键，如 shop/deployment/api */
  subject: string;
  /** 后端由 subject 解析出的类型 */
  subjectType: string;
  /** 领头实体的键 */
  rootCause: string;
}

export interface IncidentSubject extends EntityKeyParts {
  /** 主体键本身：用来打开资源详情、查这个主体的事件史 */
  key: string;
}

/**
 * 事件的主体：类型 + 命名空间 + 名称。
 *
 * 类型以后端给的 subjectType 为准，没有给时才用从键里拆出来的。
 * 主体键为空（不该出现；旧数据或接口没带）时退回到根因实体，页面不至于显示空白。
 */
export function incidentSubject(incident: IncidentSubjectSource): IncidentSubject {
  const key = incident.subject || incident.rootCause;
  const parsed = parseEntityKey(key);
  return { ...parsed, key, type: (incident.subject && incident.subjectType) || parsed.type };
}
