/**
 * Mock data for extra K8s resources (cluster: zgmf-x10a)
 *
 * StatefulSet, DaemonSet, Job, CronJob, PV, PVC, NetworkPolicy,
 * ResourceQuota, LimitRange, ServiceAccount, ConfigMap, Secret
 */

import type { StatefulSetListItem, DaemonSetListItem } from "@/api/workload";
import type {
  JobItem, CronJobItem, PVItem, PVCItem,
  NetworkPolicyItem, ResourceQuotaItem, LimitRangeItem, ServiceAccountItem,
} from "@/api/cluster-resources";
import type { ConfigMapDTO, SecretDTO } from "@/types/cluster";

// --- StatefulSets ---

export const MOCK_STATEFULSETS: StatefulSetListItem[] = [{
  name: "redis", namespace: "geass", replicas: 1, ready: 1,
  current: 1, updated: 1, available: 1,
  createdAt: "2026-01-10T08:00:00Z", age: "42d", serviceName: "redis-headless",
}];

// --- DaemonSets ---

export const MOCK_DAEMONSETS: DaemonSetListItem[] = [
  {
    name: "kube-proxy", namespace: "kube-system",
    desired: 3, current: 3, ready: 3, available: 3, misscheduled: 0,
    createdAt: "2025-09-15T06:30:00Z", age: "159d",
  },
  {
    name: "linkerd-proxy", namespace: "linkerd",
    desired: 3, current: 3, ready: 3, available: 3, misscheduled: 0,
    createdAt: "2025-11-01T10:00:00Z", age: "112d",
  },
];

// --- Jobs ---

export const MOCK_JOBS: JobItem[] = [
  {
    name: "db-migration-001", namespace: "geass",
    active: 0, succeeded: 1, failed: 0, complete: true,
    startTime: "2026-02-18T03:00:00Z", finishTime: "2026-02-18T03:02:15Z",
    createdAt: "2026-02-18T02:59:50Z", age: "3d",
  },
  {
    name: "backup-20260220", namespace: "geass",
    active: 0, succeeded: 1, failed: 0, complete: true,
    startTime: "2026-02-20T03:00:00Z", finishTime: "2026-02-20T03:10:42Z",
    createdAt: "2026-02-20T02:59:55Z", age: "1d",
  },
];

// --- CronJobs ---

export const MOCK_CRONJOBS: CronJobItem[] = [
  {
    name: "log-cleanup", namespace: "kube-system",
    schedule: "0 2 * * *", suspend: false, activeJobs: 0,
    lastScheduleTime: "2026-02-21T02:00:00Z", lastSuccessfulTime: "2026-02-21T02:01:30Z",
    createdAt: "2025-12-01T09:00:00Z", age: "82d",
  },
  {
    name: "db-backup", namespace: "geass",
    schedule: "0 3 * * 0", suspend: false, activeJobs: 0,
    lastScheduleTime: "2026-02-16T03:00:00Z", lastSuccessfulTime: "2026-02-16T03:10:42Z",
    createdAt: "2026-01-05T12:00:00Z", age: "47d",
  },
];

// --- PersistentVolumes ---

export const MOCK_PVS: PVItem[] = [
  {
    name: "nfs-pv-data", capacity: "100Gi", phase: "Bound", storageClass: "nfs",
    accessModes: ["ReadWriteMany"], reclaimPolicy: "Retain",
    createdAt: "2025-10-20T14:00:00Z", age: "124d",
  },
  {
    name: "nfs-pv-media", capacity: "50Gi", phase: "Bound", storageClass: "nfs",
    accessModes: ["ReadWriteMany"], reclaimPolicy: "Retain",
    createdAt: "2025-11-10T09:30:00Z", age: "103d",
  },
];

// --- PersistentVolumeClaims ---

export const MOCK_PVCS: PVCItem[] = [
  {
    name: "nfs-pvc-data", namespace: "geass", phase: "Bound",
    volumeName: "nfs-pv-data", storageClass: "nfs", accessModes: ["ReadWriteMany"],
    requestedCapacity: "100Gi", actualCapacity: "100Gi",
    createdAt: "2025-10-20T14:05:00Z", age: "124d",
  },
  {
    name: "nfs-pvc-media", namespace: "geass", phase: "Bound",
    volumeName: "nfs-pv-media", storageClass: "nfs", accessModes: ["ReadWriteMany"],
    requestedCapacity: "50Gi", actualCapacity: "50Gi",
    createdAt: "2025-11-10T09:35:00Z", age: "103d",
  },
];

// --- NetworkPolicies ---

export const MOCK_NETWORK_POLICIES: NetworkPolicyItem[] = [
  {
    name: "geass-default-deny", namespace: "geass",
    podSelector: "", policyTypes: ["Ingress", "Egress"],
    ingressRuleCount: 0, egressRuleCount: 0,
    createdAt: "2026-01-15T11:00:00Z", age: "37d",
  },
  {
    name: "geass-allow-traefik", namespace: "geass",
    podSelector: "app=geass-web", policyTypes: ["Ingress"],
    ingressRuleCount: 1, egressRuleCount: 0,
    createdAt: "2026-01-15T11:05:00Z", age: "37d",
  },
];

// --- ResourceQuotas ---

export const MOCK_RESOURCE_QUOTAS: ResourceQuotaItem[] = [{
  name: "geass-quota", namespace: "geass",
  hard: { cpu: "8", memory: "16Gi", pods: "20" },
  used: { cpu: "3200m", memory: "6Gi", pods: "12" },
  createdAt: "2026-01-10T08:00:00Z", age: "42d",
}];

// --- LimitRanges ---

export const MOCK_LIMIT_RANGES: LimitRangeItem[] = [{
  name: "geass-limits", namespace: "geass",
  items: [{
    type: "Container",
    min: { cpu: "50m", memory: "64Mi" },
    max: { cpu: "500m", memory: "512Mi" },
    default: { cpu: "200m", memory: "256Mi" },
    defaultRequest: { cpu: "100m", memory: "128Mi" },
  }],
  createdAt: "2026-01-10T08:05:00Z", age: "42d",
}];

// --- ServiceAccounts ---

export const MOCK_SERVICE_ACCOUNTS: ServiceAccountItem[] = [
  {
    name: "default", namespace: "default",
    secretsCount: 0, imagePullSecretsCount: 0, automountServiceAccountToken: true,
    createdAt: "2025-09-15T06:00:00Z", age: "159d",
  },
  {
    name: "default", namespace: "kube-system",
    secretsCount: 0, imagePullSecretsCount: 0, automountServiceAccountToken: true,
    createdAt: "2025-09-15T06:00:00Z", age: "159d",
  },
  {
    name: "default", namespace: "geass",
    secretsCount: 1, imagePullSecretsCount: 0, automountServiceAccountToken: true,
    createdAt: "2025-10-01T10:00:00Z", age: "143d",
  },
  {
    name: "traefik", namespace: "traefik",
    secretsCount: 1, imagePullSecretsCount: 0, automountServiceAccountToken: true,
    createdAt: "2025-10-10T08:00:00Z", age: "134d",
  },
  {
    name: "linkerd", namespace: "linkerd",
    secretsCount: 1, imagePullSecretsCount: 0, automountServiceAccountToken: false,
    createdAt: "2025-11-01T10:00:00Z", age: "112d",
  },
];

// --- ConfigMaps ---

export const MOCK_CONFIGMAPS: ConfigMapDTO[] = [
  {
    name: "coredns", namespace: "kube-system",
    createdAt: "2025-09-15T06:00:00Z", age: "159d", immutable: false,
    labels: { "k8s-app": "kube-dns" },
    keys: 1, binaryKeys: 0, totalSizeBytes: 924, binaryTotalSizeBytes: 0,
  },
  {
    name: "geass-config", namespace: "geass",
    createdAt: "2025-10-01T10:30:00Z", age: "143d", immutable: false,
    labels: { app: "geass" },
    keys: 3, binaryKeys: 0, totalSizeBytes: 2048, binaryTotalSizeBytes: 0,
  },
  {
    name: "traefik-config", namespace: "traefik",
    createdAt: "2025-10-10T08:15:00Z", age: "134d", immutable: false,
    labels: { app: "traefik" },
    keys: 2, binaryKeys: 0, totalSizeBytes: 1536, binaryTotalSizeBytes: 0,
  },
];

// --- Secrets (values never exposed — keys only) ---

export const MOCK_SECRETS: SecretDTO[] = [
  {
    name: "default-token-xxx", namespace: "default",
    type: "kubernetes.io/service-account-token",
    dataKeys: ["ca.crt", "namespace", "token"],
    createdAt: "2025-09-15T06:00:00Z", age: "159d",
  },
  {
    name: "geass-tls", namespace: "geass",
    type: "kubernetes.io/tls",
    dataKeys: ["tls.crt", "tls.key"],
    createdAt: "2026-01-20T14:00:00Z", age: "32d",
  },
  {
    name: "traefik-tls", namespace: "traefik",
    type: "kubernetes.io/tls",
    dataKeys: ["tls.crt", "tls.key"],
    createdAt: "2025-10-10T08:30:00Z", age: "134d",
  },
];
