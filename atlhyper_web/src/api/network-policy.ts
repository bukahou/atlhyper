/**
 * NetworkPolicy API
 *
 * 后端已完成 model_v2 → model 扁平化转换，前端直接使用
 */

import { get } from "./request";

// ============================================================
// 类型定义（匹配后端 model 响应）
// ============================================================

export interface NetworkPolicyItem {
  name: string;
  namespace: string;
  podSelector: string;
  policyTypes: string[];
  ingressRuleCount: number;
  egressRuleCount: number;
  createdAt: string;
  age: string;
}

export interface NetworkPolicyPeer {
  type: string;
  selector?: string;
  cidr?: string;
  except?: string[];
}

export interface NetworkPolicyPort {
  protocol: string;
  port: string;
  endPort?: number;
}

export interface NetworkPolicyRule {
  peers?: NetworkPolicyPeer[];
  ports?: NetworkPolicyPort[];
}

export interface NetworkPolicyDetail {
  name: string;
  namespace: string;
  podSelector: string;
  policyTypes: string[];
  ingressRuleCount: number;
  egressRuleCount: number;
  ingressRules?: NetworkPolicyRule[];
  egressRules?: NetworkPolicyRule[];
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
}

// ============================================================
// 响应类型（内部使用）
// ============================================================

interface ListResponse<T> {
  message: string;
  data: T[];
  total: number;
}

interface DetailResponse<T> {
  message: string;
  data: T;
}

// ============================================================
// API 查询参数（内部使用）
// ============================================================

interface ClusterResourceParams {
  cluster_id: string;
  namespace?: string;
}

// ============================================================
// API Functions
// ============================================================

export function getNetworkPolicyList(params: ClusterResourceParams) {
  return get<ListResponse<NetworkPolicyItem>>("/api/v2/network-policies", params);
}

export function getNetworkPolicyDetail(params: { ClusterID: string; Namespace: string; Name: string }) {
  return get<DetailResponse<NetworkPolicyDetail>>(
    `/api/v2/network-policies/${encodeURIComponent(params.Name)}`,
    { cluster_id: params.ClusterID, namespace: params.Namespace }
  );
}
