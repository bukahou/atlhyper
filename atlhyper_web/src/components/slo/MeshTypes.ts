export interface MeshTabTranslations {
  serviceTopology: string;
  meshOverview: string;
  service: string;
  rps: string;
  p95Latency: string;
  errorRate: string;
  mtls: string;
  status: string;
  healthy: string;
  warning: string;
  critical: string;
  inbound: string;
  outbound: string;
  noCallData: string;
  callRelation: string;
  p50Latency: string;
  p99Latency: string;
  totalRequests: string;
  avgLatency: string;
  statusCodeBreakdown: string;
  latencyDistribution: string;
  requests: string;
  loading: string;
}

// Time range display label
export function timeRangeLabel(tr: string): string {
  switch (tr) {
    case "1d": return "24h";
    case "7d": return "7d";
    case "30d": return "30d";
    default: return tr;
  }
}
