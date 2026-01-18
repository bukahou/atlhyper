/**
 * Service API
 *
 * 适配 Master V2 API（嵌套结构）
 */

import { get } from "./request";
import type { ServiceOverview, ServiceDetail, ServiceItem } from "@/types/cluster";

// ============================================================
// 查询参数类型
// ============================================================

interface ServiceListParams {
  cluster_id: string;
  namespace?: string;
  limit?: number;
  offset?: number;
}

// ============================================================
// 响应类型（后端返回嵌套结构）
// ============================================================

// 后端返回的 Service 格式（与 model_v2.Service 一致）
interface ServiceApiItem {
  summary: {
    name: string;
    namespace: string;
    type: string;
    createdAt: string;
    age: string;
    portsCount: number;
    hasSelector: boolean;
    badges?: string[];
    clusterIP?: string;
    externalName?: string;
  };
  spec: {
    type: string;
    sessionAffinity?: string;
    sessionAffinityTimeoutSeconds?: number;
    externalTrafficPolicy?: string;
    internalTrafficPolicy?: string;
    ipFamilies?: string[];
    ipFamilyPolicy?: string;
    clusterIPs?: string[];
    externalIPs?: string[];
    loadBalancerClass?: string;
    loadBalancerSourceRanges?: string[];
    publishNotReadyAddresses?: boolean;
    allocateLoadBalancerNodePorts?: boolean;
    healthCheckNodePort?: number;
    externalName?: string;
  };
  ports?: {
    name?: string;
    protocol: string;
    port: number;
    targetPort: string;
    nodePort?: number;
    appProtocol?: string;
  }[];
  selector?: Record<string, string>;
  network: {
    clusterIPs?: string[];
    externalIPs?: string[];
    loadBalancerIngress?: string[];
    ipFamilies?: string[];
    ipFamilyPolicy?: string;
    externalTrafficPolicy?: string;
    internalTrafficPolicy?: string;
  };
  backends?: {
    summary: {
      ready: number;
      notReady: number;
      total: number;
      slices?: number;
      updated?: string;
    };
    ports?: {
      name?: string;
      port: number;
      protocol: string;
      appProtocol?: string;
    }[];
    endpoints?: {
      address: string;
      ready: boolean;
      nodeName?: string;
      zone?: string;
      targetRef?: {
        kind?: string;
        namespace?: string;
        name?: string;
        uid?: string;
      };
    }[];
  };
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

interface ServiceListResponse {
  message: string;
  data: ServiceApiItem[];
  total: number;
}

interface ServiceDetailResponse {
  message: string;
  data: ServiceApiItem;
}

// ============================================================
// API 方法
// ============================================================

/**
 * 获取 Service 列表
 * GET /api/v2/services?cluster_id=xxx&namespace=xxx
 */
export function getServiceList(params: ServiceListParams) {
  return get<ServiceListResponse>("/api/v2/services", params);
}

/**
 * 获取 Service 详情
 * GET /api/v2/services/{name}?cluster_id=xxx&namespace=xxx
 */
export async function getServiceDetail(data: { ClusterID: string; Namespace: string; Name: string }) {
  const response = await get<ServiceDetailResponse>(
    `/api/v2/services/${encodeURIComponent(data.Name)}`,
    {
      cluster_id: data.ClusterID,
      namespace: data.Namespace,
    }
  );

  const apiSvc = response.data.data;
  if (!apiSvc) {
    throw new Error("Service not found");
  }

  // 转换为前端详情格式
  const detail = transformToServiceDetail(apiSvc);

  return {
    ...response,
    data: {
      data: detail,
    },
  };
}

/**
 * 获取 Service 概览（包含统计卡片和列表）
 */
export async function getServiceOverview(data: { ClusterID: string }) {
  const response = await getServiceList({ cluster_id: data.ClusterID });
  const services = response.data.data || [];
  const overview = transformToServiceOverview(services);

  return {
    ...response,
    data: {
      data: overview,
    },
  };
}

// ============================================================
// 数据转换
// ============================================================

/**
 * 将后端 API 返回的 Service 数据转换为列表项格式
 */
function transformServiceItem(apiItem: ServiceApiItem): ServiceItem {
  const summary = apiItem.summary;
  const ports = apiItem.ports || [];

  // 格式化端口显示
  const portsStr = ports.length > 0
    ? ports.map(p => {
        const nodePort = p.nodePort ? `:${p.nodePort}` : "";
        return `${p.port}${nodePort}/${p.protocol}→${p.targetPort}`;
      }).join(", ")
    : "-";

  // 格式化选择器显示
  const selectorStr = apiItem.selector
    ? Object.entries(apiItem.selector).map(([k, v]) => `${k}=${v}`).join(", ")
    : "-";

  return {
    name: summary.name,
    namespace: summary.namespace,
    type: summary.type,
    clusterIP: summary.clusterIP || "",
    ports: portsStr,
    protocol: ports[0]?.protocol || "TCP",
    selector: selectorStr,
    createdAt: summary.createdAt,
  };
}

/**
 * 将 Service 列表转换为 ServiceOverview 格式
 */
function transformToServiceOverview(apiServices: ServiceApiItem[]): ServiceOverview {
  const services = apiServices.map(transformServiceItem);
  let externalServices = 0;
  let internalServices = 0;
  let headlessServices = 0;

  for (const api of apiServices) {
    const type = api.summary.type?.toLowerCase() || "";
    const clusterIP = api.summary.clusterIP || "";

    if (type === "loadbalancer" || type === "nodeport") {
      externalServices++;
    } else if (type === "clusterip" && clusterIP === "None") {
      headlessServices++;
    } else {
      internalServices++;
    }
  }

  return {
    cards: {
      totalServices: services.length,
      externalServices,
      internalServices,
      headlessServices,
    },
    rows: services,
  };
}

/**
 * 将后端 Service 转换为详情格式
 */
function transformToServiceDetail(apiSvc: ServiceApiItem): ServiceDetail {
  const summary = apiSvc.summary;
  const spec = apiSvc.spec;
  const network = apiSvc.network;
  const backends = apiSvc.backends;

  return {
    // 基本信息
    name: summary.name,
    namespace: summary.namespace,
    type: summary.type,
    createdAt: summary.createdAt,
    age: summary.age,

    // 选择器 & 端口
    selector: apiSvc.selector,
    ports: apiSvc.ports?.map(p => ({
      name: p.name,
      protocol: p.protocol,
      port: p.port,
      targetPort: p.targetPort,
      nodePort: p.nodePort,
      appProtocol: p.appProtocol,
    })),

    // 网络信息
    clusterIPs: network.clusterIPs,
    externalIPs: network.externalIPs,
    loadBalancerIngress: network.loadBalancerIngress,

    // Spec
    sessionAffinity: spec.sessionAffinity,
    sessionAffinityTimeoutSeconds: spec.sessionAffinityTimeoutSeconds,
    externalTrafficPolicy: spec.externalTrafficPolicy || network.externalTrafficPolicy,
    internalTrafficPolicy: spec.internalTrafficPolicy || network.internalTrafficPolicy,
    ipFamilies: spec.ipFamilies || network.ipFamilies,
    ipFamilyPolicy: spec.ipFamilyPolicy || network.ipFamilyPolicy,
    loadBalancerClass: spec.loadBalancerClass,
    loadBalancerSourceRanges: spec.loadBalancerSourceRanges,
    allocateLoadBalancerNodePorts: spec.allocateLoadBalancerNodePorts,
    healthCheckNodePort: spec.healthCheckNodePort,
    externalName: spec.externalName || summary.externalName,

    // 端点聚合
    backends: backends ? {
      ready: backends.summary.ready,
      notReady: backends.summary.notReady,
      total: backends.summary.total,
      slices: backends.summary.slices,
      updated: backends.summary.updated,
      ports: backends.ports,
      endpoints: backends.endpoints?.map(ep => ({
        address: ep.address,
        ready: ep.ready,
        nodeName: ep.nodeName,
        zone: ep.zone,
        targetRef: ep.targetRef,
      })),
    } : undefined,

    // 徽标
    badges: summary.badges,
  };
}
