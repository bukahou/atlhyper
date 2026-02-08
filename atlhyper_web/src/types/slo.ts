/**
 * SLO 监控相关类型定义
 * 与后端 atlhyper_master_v2/model/slo.go 保持一致
 */

// SLO 状态
export type SLOStatus = "healthy" | "warning" | "critical" | "unknown";

// 趋势方向
export type SLOTrend = "up" | "down" | "stable";

// SLO 指标（对应 model.SLOMetrics）
export interface SLOMetrics {
  availability: number;       // 可用性 (0-100)
  p95_latency: number;        // P95 延迟 (ms)
  p99_latency: number;        // P99 延迟 (ms)
  error_rate: number;         // 错误率 (0-100)
  requests_per_sec: number;   // 每秒请求数
  total_requests: number;     // 总请求数
}

// SLO 目标规格（对应 model.SLOTargetSpec）
export interface SLOTargetSpec {
  availability: number;       // 目标可用性
  p95_latency: number;        // 目标 P95 延迟 (ms)
}

// 域名 SLO（对应 model.DomainSLO）
export interface DomainSLO {
  host: string;
  ingress_name: string;
  ingress_class: string;
  namespace: string;
  tls: boolean;
  targets: Record<string, SLOTargetSpec>;  // "1d", "7d", "30d"
  current: SLOMetrics | null;
  previous?: SLOMetrics | null;
  error_budget_remaining: number;
  status: SLOStatus;
  trend: SLOTrend;
}

// SLO 汇总（对应 model.SLOSummary）
export interface SLOSummary {
  total_services: number;
  total_domains: number;
  healthy_count: number;
  warning_count: number;
  critical_count: number;
  avg_availability: number;
  avg_error_budget: number;
  total_rps: number;
}

// 域名 SLO 列表响应（对应 model.SLODomainsResponse）
export interface DomainSLOListResponse {
  domains: DomainSLO[];
  summary: SLOSummary;
}

// 域名 SLO 详情（复用 DomainSLO）
export type DomainSLODetail = DomainSLO;

// 历史数据点（对应 model.SLODomainHistoryItem）
export interface SLOHistoryPoint {
  timestamp: string;
  availability: number;
  p95_latency: number;
  p99_latency: number;
  rps: number;
  error_rate: number;
}

// 域名历史数据响应（对应 model.SLODomainHistoryResponse）
export interface DomainSLOHistoryResponse {
  host: string;
  history: SLOHistoryPoint[];
}

// SLO 目标配置
export interface SLOTarget {
  id?: number;
  cluster_id: string;
  host: string;
  time_range: string;
  availability_target: number;
  p95_latency_target: number;
  created_at?: string;
  updated_at?: string;
}

// 状态变更历史项
export interface SLOStatusHistoryItem {
  host: string;
  time_range: string;
  old_status: SLOStatus;
  new_status: SLOStatus;
  availability: number;
  p95_latency: number;
  error_budget_remaining: number;
  changed_at: string;
}

// 状态历史响应
export type SLOStatusHistoryResponse = SLOStatusHistoryItem[];

// ==================== V2 API 类型（按真实域名分组） ====================

// 后端服务级别 SLO（对应 model.ServiceSLO）
// 注意：Metrics 数据是按 service 级别聚合的，不是按 path 级别
export interface ServiceSLO {
  service_key: string;             // Traefik service key (namespace-name-port@kubernetes)
  service_name: string;            // 服务名称
  service_port: number;            // 服务端口
  namespace: string;               // 命名空间
  paths: string[];                 // 使用该服务的路径列表（仅展示用，共享同一份 metrics）
  ingress_name: string;            // IngressRoute/Ingress 名称
  current: SLOMetrics | null;      // 当前周期指标
  previous?: SLOMetrics | null;    // 上一周期指标
  targets?: Record<string, SLOTargetSpec>;  // 目标配置
  status: SLOStatus;               // 状态
  error_budget_remaining: number;  // 剩余错误预算
}

// 域名级别 SLO（对应 model.DomainSLOResponseV2）
export interface DomainSLOV2 {
  domain: string;                  // 真实域名（如 example.com）
  tls: boolean;                    // 是否启用 TLS
  services: ServiceSLO[];          // 该域名下的所有后端服务
  summary: SLOMetrics | null;      // 域名级别汇总指标
  targets?: Record<string, SLOTargetSpec>;  // 目标配置 ("1d"/"7d"/"30d")
  status: SLOStatus;               // 域名状态
  error_budget_remaining: number;  // 域名剩余错误预算
}

// V2 域名列表响应（对应 model.SLODomainsResponseV2）
export interface DomainSLOListResponseV2 {
  domains: DomainSLOV2[];
  summary: SLOSummary;
}

// ==================== 延迟分布 API 类型 ====================

// 延迟分布桶
export interface LatencyBucket {
  le: number;       // 上界 (ms)
  count: number;    // 该桶内的请求数
}

// HTTP 方法分布
export interface MethodBreakdown {
  method: string;   // GET, POST, PUT, DELETE, OTHER
  count: number;
}

// 状态码分布
export interface StatusCodeBreakdown {
  code: string;     // "2xx", "3xx", "4xx", "5xx"
  count: number;
}

// 延迟分布响应
export interface LatencyDistributionResponse {
  domain: string;
  total_requests: number;
  p50_latency_ms: number;
  p95_latency_ms: number;
  p99_latency_ms: number;
  avg_latency_ms: number;
  buckets: LatencyBucket[];
  methods: MethodBreakdown[];
  status_codes: StatusCodeBreakdown[];
}

// 延迟分布请求参数
export interface SLOLatencyParams {
  clusterId: string;
  domain: string;
  timeRange?: string;
}

// API 请求参数
export interface SLODomainsParams {
  clusterId?: string;
  timeRange?: string;  // "1d" | "7d" | "30d"
}

export interface SLODomainDetailParams {
  clusterId: string;
  host: string;
  timeRange?: string;
}

export interface SLODomainHistoryParams {
  clusterId: string;
  host: string;
  timeRange?: string;
}

export interface SLOTargetCreateParams {
  clusterId: string;
  host: string;
  timeRange: string;
  availabilityTarget: number;
  p95LatencyTarget: number;
}

export interface SLOStatusHistoryParams {
  clusterId?: string;
  host?: string;
  limit?: number;
}
