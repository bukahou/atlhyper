/**
 * Mock data for 8 K8s resources (Job, CronJob, PV, PVC, NetworkPolicy, ResourceQuota, LimitRange, ServiceAccount)
 * Data structures match Agent model_v2 JSON output
 */

// ============================================================
// Job
// ============================================================

export interface JobItem {
  name: string;
  namespace: string;
  active: number;
  succeeded: number;
  failed: number;
  completions: number;
  parallelism: number;
  startTime: string;
  completionTime: string;
  duration: string;
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
}

export function mockJobs(): JobItem[] {
  return [
    { name: "data-migration-v2", namespace: "default", active: 0, succeeded: 1, failed: 0, completions: 1, parallelism: 1, startTime: "2026-02-13T10:00:00Z", completionTime: "2026-02-13T10:05:32Z", duration: "5m32s", createdAt: "2026-02-13T10:00:00Z", age: "1d", labels: { app: "migration" } },
    { name: "backup-db-20260214", namespace: "default", active: 1, succeeded: 0, failed: 0, completions: 1, parallelism: 1, startTime: "2026-02-14T02:00:00Z", completionTime: "", duration: "", createdAt: "2026-02-14T02:00:00Z", age: "12h", labels: { app: "backup" } },
    { name: "report-gen-weekly", namespace: "analytics", active: 0, succeeded: 3, failed: 0, completions: 3, parallelism: 3, startTime: "2026-02-13T08:00:00Z", completionTime: "2026-02-13T08:12:45Z", duration: "12m45s", createdAt: "2026-02-13T08:00:00Z", age: "1d", labels: { app: "reports" } },
    { name: "cleanup-temp-files", namespace: "system", active: 0, succeeded: 1, failed: 0, completions: 1, parallelism: 1, startTime: "2026-02-14T00:00:00Z", completionTime: "2026-02-14T00:01:12Z", duration: "1m12s", createdAt: "2026-02-14T00:00:00Z", age: "14h" },
    { name: "etl-pipeline-run", namespace: "data", active: 0, succeeded: 0, failed: 2, completions: 1, parallelism: 1, startTime: "2026-02-14T06:00:00Z", completionTime: "", duration: "", createdAt: "2026-02-14T06:00:00Z", age: "8h", labels: { app: "etl", team: "data-eng" } },
    { name: "index-rebuild", namespace: "search", active: 0, succeeded: 1, failed: 0, completions: 1, parallelism: 1, startTime: "2026-02-12T22:00:00Z", completionTime: "2026-02-12T22:45:00Z", duration: "45m", createdAt: "2026-02-12T22:00:00Z", age: "2d" },
  ];
}

// ============================================================
// CronJob
// ============================================================

export interface CronJobItem {
  name: string;
  namespace: string;
  schedule: string;
  suspend: boolean;
  activeJobs: number;
  lastScheduleTime: string;
  lastSuccessfulTime: string;
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
}

export function mockCronJobs(): CronJobItem[] {
  return [
    { name: "backup-daily", namespace: "default", schedule: "0 2 * * *", suspend: false, activeJobs: 0, lastScheduleTime: "2026-02-14T02:00:00Z", lastSuccessfulTime: "2026-02-14T02:05:00Z", createdAt: "2026-01-01T00:00:00Z", age: "44d", labels: { app: "backup" } },
    { name: "log-rotate", namespace: "system", schedule: "0 0 * * *", suspend: false, activeJobs: 0, lastScheduleTime: "2026-02-14T00:00:00Z", lastSuccessfulTime: "2026-02-14T00:01:00Z", createdAt: "2025-12-15T00:00:00Z", age: "61d" },
    { name: "report-weekly", namespace: "analytics", schedule: "0 8 * * 1", suspend: false, activeJobs: 0, lastScheduleTime: "2026-02-10T08:00:00Z", lastSuccessfulTime: "2026-02-10T08:12:00Z", createdAt: "2026-01-10T00:00:00Z", age: "35d", labels: { app: "reports" } },
    { name: "cleanup-old-data", namespace: "data", schedule: "0 3 * * 0", suspend: false, activeJobs: 0, lastScheduleTime: "2026-02-09T03:00:00Z", lastSuccessfulTime: "2026-02-09T03:10:00Z", createdAt: "2026-01-05T00:00:00Z", age: "40d" },
    { name: "health-check", namespace: "monitoring", schedule: "*/5 * * * *", suspend: false, activeJobs: 1, lastScheduleTime: "2026-02-14T13:55:00Z", lastSuccessfulTime: "2026-02-14T13:50:00Z", createdAt: "2025-11-01T00:00:00Z", age: "105d" },
    { name: "cert-renew", namespace: "cert-manager", schedule: "0 0 1 * *", suspend: true, activeJobs: 0, lastScheduleTime: "2026-02-01T00:00:00Z", lastSuccessfulTime: "2026-02-01T00:02:00Z", createdAt: "2025-10-01T00:00:00Z", age: "136d" },
  ];
}

// ============================================================
// PersistentVolume
// ============================================================

export interface PVItem {
  name: string;
  capacity: string;
  accessModes: string[];
  reclaimPolicy: string;
  status: string;
  storageClass: string;
  claimRef: string;
  volumeMode: string;
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
}

export function mockPVs(): PVItem[] {
  return [
    { name: "pv-data-001", capacity: "100Gi", accessModes: ["ReadWriteOnce"], reclaimPolicy: "Retain", status: "Bound", storageClass: "standard", claimRef: "default/data-pvc-001", volumeMode: "Filesystem", createdAt: "2025-12-01T00:00:00Z", age: "75d" },
    { name: "pv-logs-001", capacity: "50Gi", accessModes: ["ReadWriteMany"], reclaimPolicy: "Delete", status: "Bound", storageClass: "nfs", claimRef: "logging/logs-pvc", volumeMode: "Filesystem", createdAt: "2026-01-15T00:00:00Z", age: "30d" },
    { name: "pv-backup-001", capacity: "200Gi", accessModes: ["ReadWriteOnce"], reclaimPolicy: "Retain", status: "Available", storageClass: "standard", claimRef: "", volumeMode: "Filesystem", createdAt: "2026-02-01T00:00:00Z", age: "13d" },
    { name: "pv-mysql-data", capacity: "500Gi", accessModes: ["ReadWriteOnce"], reclaimPolicy: "Retain", status: "Bound", storageClass: "ssd", claimRef: "database/mysql-data-pvc", volumeMode: "Filesystem", createdAt: "2025-10-01T00:00:00Z", age: "136d" },
    { name: "pv-redis-001", capacity: "10Gi", accessModes: ["ReadWriteOnce"], reclaimPolicy: "Delete", status: "Released", storageClass: "standard", claimRef: "", volumeMode: "Filesystem", createdAt: "2025-11-15T00:00:00Z", age: "91d" },
    { name: "pv-shared-nfs", capacity: "1Ti", accessModes: ["ReadWriteMany", "ReadOnlyMany"], reclaimPolicy: "Retain", status: "Bound", storageClass: "nfs", claimRef: "default/shared-data", volumeMode: "Filesystem", createdAt: "2025-09-01T00:00:00Z", age: "166d" },
  ];
}

// ============================================================
// PersistentVolumeClaim
// ============================================================

export interface PVCItem {
  name: string;
  namespace: string;
  status: string;
  volume: string;
  capacity: string;
  requestedCapacity: string;
  accessModes: string[];
  storageClass: string;
  volumeMode: string;
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
}

export function mockPVCs(): PVCItem[] {
  return [
    { name: "data-pvc-001", namespace: "default", status: "Bound", volume: "pv-data-001", capacity: "100Gi", requestedCapacity: "100Gi", accessModes: ["ReadWriteOnce"], storageClass: "standard", volumeMode: "Filesystem", createdAt: "2025-12-01T00:00:00Z", age: "75d" },
    { name: "logs-pvc", namespace: "logging", status: "Bound", volume: "pv-logs-001", capacity: "50Gi", requestedCapacity: "50Gi", accessModes: ["ReadWriteMany"], storageClass: "nfs", volumeMode: "Filesystem", createdAt: "2026-01-15T00:00:00Z", age: "30d" },
    { name: "mysql-data-pvc", namespace: "database", status: "Bound", volume: "pv-mysql-data", capacity: "500Gi", requestedCapacity: "500Gi", accessModes: ["ReadWriteOnce"], storageClass: "ssd", volumeMode: "Filesystem", createdAt: "2025-10-01T00:00:00Z", age: "136d" },
    { name: "cache-pvc", namespace: "default", status: "Pending", volume: "", capacity: "", requestedCapacity: "20Gi", accessModes: ["ReadWriteOnce"], storageClass: "fast", volumeMode: "Filesystem", createdAt: "2026-02-14T10:00:00Z", age: "4h" },
    { name: "shared-data", namespace: "default", status: "Bound", volume: "pv-shared-nfs", capacity: "1Ti", requestedCapacity: "1Ti", accessModes: ["ReadWriteMany"], storageClass: "nfs", volumeMode: "Filesystem", createdAt: "2025-09-01T00:00:00Z", age: "166d" },
    { name: "temp-storage", namespace: "batch", status: "Lost", volume: "pv-temp-gone", capacity: "5Gi", requestedCapacity: "5Gi", accessModes: ["ReadWriteOnce"], storageClass: "standard", volumeMode: "Filesystem", createdAt: "2026-02-10T00:00:00Z", age: "4d" },
  ];
}

// ============================================================
// NetworkPolicy
// ============================================================

export interface NetworkPolicyItem {
  name: string;
  namespace: string;
  policyTypes: string[];
  ingressRules: number;
  egressRules: number;
  podSelector: Record<string, string>;
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
}

export function mockNetworkPolicies(): NetworkPolicyItem[] {
  return [
    { name: "deny-all-ingress", namespace: "production", policyTypes: ["Ingress"], ingressRules: 0, egressRules: 0, podSelector: {}, createdAt: "2025-12-01T00:00:00Z", age: "75d" },
    { name: "allow-web-traffic", namespace: "production", policyTypes: ["Ingress"], ingressRules: 2, egressRules: 0, podSelector: { app: "web" }, createdAt: "2025-12-01T00:00:00Z", age: "75d" },
    { name: "api-egress-policy", namespace: "production", policyTypes: ["Egress"], ingressRules: 0, egressRules: 3, podSelector: { app: "api" }, createdAt: "2026-01-10T00:00:00Z", age: "35d" },
    { name: "db-access-policy", namespace: "database", policyTypes: ["Ingress", "Egress"], ingressRules: 1, egressRules: 1, podSelector: { app: "mysql" }, createdAt: "2025-11-15T00:00:00Z", age: "91d" },
    { name: "monitoring-access", namespace: "monitoring", policyTypes: ["Ingress"], ingressRules: 3, egressRules: 0, podSelector: { app: "prometheus" }, createdAt: "2026-01-20T00:00:00Z", age: "25d" },
    { name: "default-deny-all", namespace: "staging", policyTypes: ["Ingress", "Egress"], ingressRules: 0, egressRules: 0, podSelector: {}, createdAt: "2026-02-01T00:00:00Z", age: "13d" },
  ];
}

// ============================================================
// ResourceQuota
// ============================================================

export interface ResourceQuotaItem {
  name: string;
  namespace: string;
  hard: Record<string, string>;
  used: Record<string, string>;
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
}

export function mockResourceQuotas(): ResourceQuotaItem[] {
  return [
    { name: "compute-quota", namespace: "production", hard: { "requests.cpu": "10", "requests.memory": "20Gi", "limits.cpu": "20", "limits.memory": "40Gi", pods: "50" }, used: { "requests.cpu": "6.5", "requests.memory": "14Gi", "limits.cpu": "13", "limits.memory": "28Gi", pods: "32" }, createdAt: "2025-12-01T00:00:00Z", age: "75d" },
    { name: "storage-quota", namespace: "production", hard: { "requests.storage": "500Gi", persistentvolumeclaims: "10" }, used: { "requests.storage": "350Gi", persistentvolumeclaims: "7" }, createdAt: "2025-12-01T00:00:00Z", age: "75d" },
    { name: "dev-quota", namespace: "development", hard: { "requests.cpu": "4", "requests.memory": "8Gi", pods: "20" }, used: { "requests.cpu": "1.2", "requests.memory": "3Gi", pods: "8" }, createdAt: "2026-01-15T00:00:00Z", age: "30d" },
    { name: "staging-limits", namespace: "staging", hard: { "requests.cpu": "8", "requests.memory": "16Gi", "limits.cpu": "16", "limits.memory": "32Gi", pods: "30" }, used: { "requests.cpu": "4", "requests.memory": "8Gi", "limits.cpu": "8", "limits.memory": "16Gi", pods: "15" }, createdAt: "2026-01-01T00:00:00Z", age: "44d" },
  ];
}

// ============================================================
// LimitRange
// ============================================================

export interface LimitRangeLimit {
  type: string;
  max?: Record<string, string>;
  min?: Record<string, string>;
  default?: Record<string, string>;
  defaultRequest?: Record<string, string>;
}

export interface LimitRangeItem {
  name: string;
  namespace: string;
  limits: LimitRangeLimit[];
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
}

export function mockLimitRanges(): LimitRangeItem[] {
  return [
    { name: "default-limits", namespace: "production", limits: [{ type: "Container", max: { cpu: "4", memory: "8Gi" }, min: { cpu: "100m", memory: "128Mi" }, default: { cpu: "500m", memory: "512Mi" }, defaultRequest: { cpu: "200m", memory: "256Mi" } }, { type: "Pod", max: { cpu: "8", memory: "16Gi" } }], createdAt: "2025-12-01T00:00:00Z", age: "75d" },
    { name: "dev-limits", namespace: "development", limits: [{ type: "Container", max: { cpu: "2", memory: "4Gi" }, default: { cpu: "250m", memory: "256Mi" }, defaultRequest: { cpu: "100m", memory: "128Mi" } }], createdAt: "2026-01-15T00:00:00Z", age: "30d" },
    { name: "pvc-limits", namespace: "production", limits: [{ type: "PersistentVolumeClaim", max: { storage: "100Gi" }, min: { storage: "1Gi" } }], createdAt: "2025-12-15T00:00:00Z", age: "61d" },
    { name: "staging-limits", namespace: "staging", limits: [{ type: "Container", max: { cpu: "2", memory: "4Gi" }, min: { cpu: "50m", memory: "64Mi" }, default: { cpu: "300m", memory: "384Mi" }, defaultRequest: { cpu: "150m", memory: "192Mi" } }], createdAt: "2026-01-01T00:00:00Z", age: "44d" },
  ];
}

// ============================================================
// ServiceAccount
// ============================================================

export interface ServiceAccountItem {
  name: string;
  namespace: string;
  secrets: number;
  imagePullSecrets: string[];
  automountToken: boolean;
  createdAt: string;
  age: string;
  labels?: Record<string, string>;
}

export function mockServiceAccounts(): ServiceAccountItem[] {
  return [
    { name: "default", namespace: "default", secrets: 1, imagePullSecrets: [], automountToken: true, createdAt: "2025-06-01T00:00:00Z", age: "258d" },
    { name: "default", namespace: "production", secrets: 1, imagePullSecrets: ["registry-creds"], automountToken: true, createdAt: "2025-06-01T00:00:00Z", age: "258d" },
    { name: "app-deployer", namespace: "production", secrets: 2, imagePullSecrets: ["registry-creds", "gcr-key"], automountToken: false, createdAt: "2025-12-01T00:00:00Z", age: "75d", labels: { role: "deployer" } },
    { name: "monitoring-sa", namespace: "monitoring", secrets: 1, imagePullSecrets: [], automountToken: true, createdAt: "2025-11-15T00:00:00Z", age: "91d", labels: { app: "prometheus" } },
    { name: "ci-runner", namespace: "ci", secrets: 3, imagePullSecrets: ["registry-creds"], automountToken: false, createdAt: "2026-01-10T00:00:00Z", age: "35d", labels: { role: "ci" } },
    { name: "default", namespace: "kube-system", secrets: 1, imagePullSecrets: [], automountToken: true, createdAt: "2025-06-01T00:00:00Z", age: "258d" },
    { name: "cert-manager", namespace: "cert-manager", secrets: 1, imagePullSecrets: [], automountToken: true, createdAt: "2025-10-01T00:00:00Z", age: "136d" },
    { name: "default", namespace: "staging", secrets: 0, imagePullSecrets: [], automountToken: true, createdAt: "2026-01-01T00:00:00Z", age: "44d" },
  ];
}
