/**
 * Ingress API
 *
 * 适配 Master V2 API（嵌套结构）
 */

import { get } from "./request";
import type { IngressOverview, IngressDetail, IngressItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface IngressListParams {
  cluster_id: string;
  namespace?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 后端返回类型（Master V2 API 嵌套结构）
// ============================================================

// 后端 Backend 结构
interface IngressBackendApi {
  type: string;
  service?: {
    name: string;
    portName?: string;
    portNumber?: number;
  };
  resource?: {
    apiGroup?: string;
    kind: string;
    name: string;
    namespace?: string;
  };
}

// 后端 Path 结构
interface IngressPathApi {
  path: string;
  pathType: string;
  backend?: IngressBackendApi;
}

// 后端 Rule 结构
interface IngressRuleApi {
  host?: string;
  paths: IngressPathApi[];
}

// 后端 TLS 结构
interface IngressTLSApi {
  hosts?: string[];
  secretName: string;
}

// 后端返回的 Ingress 结构（嵌套）
interface IngressApiItem {
  summary: {
    name: string;
    namespace: string;
    createdAt: string;
    age: string;
    ingressClass?: string;
    hostsCount: number;
    pathsCount: number;
    tlsEnabled: boolean;
    hosts?: string[];
  };
  spec: {
    ingressClassName?: string;
    defaultBackend?: IngressBackendApi;
    rules?: IngressRuleApi[];
    tls?: IngressTLSApi[];
  };
  status: {
    loadBalancer?: string[];
  };
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

interface IngressListResponse {
  message: string;
  data: IngressApiItem[];
  total: number;
}

interface IngressDetailResponse {
  message: string;
  data: IngressApiItem;
}

// ============================================================
// 数据转换
// ============================================================

/**
 * 检查 host 是否有 TLS 配置
 */
function hasTLS(host: string, tlsConfigs?: IngressTLSApi[]): string {
  if (!tlsConfigs || tlsConfigs.length === 0) return "-";
  for (const tls of tlsConfigs) {
    if (tls.hosts?.includes(host)) {
      return tls.secretName || "Yes";
    }
  }
  return "-";
}

/**
 * 将后端 API 返回的 Ingress 数据展开为多行
 * 每个 host/path 组合生成一行（用于列表页展示）
 */
function expandIngressToRows(apiItem: IngressApiItem): IngressItem[] {
  const rows: IngressItem[] = [];
  const rules = apiItem.spec.rules || [];

  if (rules.length === 0) {
    rows.push({
      name: apiItem.summary.name,
      namespace: apiItem.summary.namespace,
      host: "-",
      path: "-",
      serviceName: "-",
      servicePort: "-",
      tls: "-",
      createdAt: apiItem.summary.createdAt,
    });
    return rows;
  }

  for (const rule of rules) {
    const host = rule.host || "-";
    const tlsStatus = hasTLS(host, apiItem.spec.tls);

    if (!rule.paths || rule.paths.length === 0) {
      rows.push({
        name: apiItem.summary.name,
        namespace: apiItem.summary.namespace,
        host,
        path: "-",
        serviceName: "-",
        servicePort: "-",
        tls: tlsStatus,
        createdAt: apiItem.summary.createdAt,
      });
    } else {
      for (const pathItem of rule.paths) {
        const backend = pathItem.backend;
        let serviceName = "-";
        let servicePort = "-";
        if (backend?.service) {
          serviceName = backend.service.name;
          servicePort = String(backend.service.portNumber || backend.service.portName || "-");
        }
        rows.push({
          name: apiItem.summary.name,
          namespace: apiItem.summary.namespace,
          host,
          path: pathItem.path || "/",
          serviceName,
          servicePort,
          tls: tlsStatus,
          createdAt: apiItem.summary.createdAt,
        });
      }
    }
  }

  return rows;
}

/**
 * 将后端 Ingress 数据转换为前端 IngressDetail 格式
 */
function transformIngressDetail(apiItem: IngressApiItem): IngressDetail {
  return {
    name: apiItem.summary.name,
    namespace: apiItem.summary.namespace,
    class: apiItem.summary.ingressClass,
    controller: undefined, // 需要从 IngressClass 资源获取
    hosts: apiItem.summary.hosts,
    tlsEnabled: apiItem.summary.tlsEnabled,
    loadBalancer: apiItem.status.loadBalancer,
    createdAt: apiItem.summary.createdAt,
    age: apiItem.summary.age,
    spec: {
      ingressClassName: apiItem.spec.ingressClassName,
      defaultBackend: apiItem.spec.defaultBackend,
      rules: apiItem.spec.rules?.map((rule) => ({
        host: rule.host,
        paths: rule.paths.map((p) => ({
          path: p.path,
          pathType: p.pathType,
          backend: p.backend || { type: "Service" },
        })),
      })),
      tls: apiItem.spec.tls?.map((t) => ({
        secretName: t.secretName,
        hosts: t.hosts,
      })),
    },
    status: {
      loadBalancer: apiItem.status.loadBalancer,
    },
    annotations: apiItem.annotations,
  };
}

/**
 * 将 Ingress 列表转换为 IngressOverview 格式
 */
function transformToIngressOverview(apiIngresses: IngressApiItem[]): IngressOverview {
  const rows: IngressItem[] = [];
  for (const apiItem of apiIngresses) {
    rows.push(...expandIngressToRows(apiItem));
  }

  // 统计
  const hostsSet = new Set<string>();
  const tlsSecretsSet = new Set<string>();
  let totalPaths = 0;

  for (const row of rows) {
    if (row.host && row.host !== "-") hostsSet.add(row.host);
    if (row.tls && row.tls !== "-") tlsSecretsSet.add(row.tls);
    if (row.path && row.path !== "-") totalPaths++;
  }

  return {
    cards: {
      totalIngresses: apiIngresses.length,
      usedHosts: hostsSet.size,
      tlsCerts: tlsSecretsSet.size,
      totalPaths,
    },
    rows,
  };
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Ingress 列表
 * GET /api/v2/ingresses?cluster_id=xxx&namespace=xxx
 */
export function getIngressList(params: IngressListParams) {
  return get<IngressListResponse>("/api/v2/ingresses", params);
}

/**
 * 获取 Ingress 详情
 * GET /api/v2/ingresses/{name}?cluster_id=xxx&namespace=xxx
 */
export async function getIngressDetail(data: { ClusterID: string; Namespace: string; Name: string }) {
  const response = await get<IngressDetailResponse>(
    `/api/v2/ingresses/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );

  const apiIngress = response.data.data;
  if (!apiIngress) {
    throw new Error("Ingress not found");
  }

  const detail = transformIngressDetail(apiIngress);

  return {
    ...response,
    data: { data: detail },
  };
}

/**
 * 获取 Ingress 概览（包含统计卡片和列表）
 */
export async function getIngressOverview(data: { ClusterID: string }) {
  const response = await getIngressList({ cluster_id: data.ClusterID });
  const ingresses = response.data.data || [];
  const overview = transformToIngressOverview(ingresses);

  return {
    ...response,
    data: {
      data: overview,
    },
  };
}
