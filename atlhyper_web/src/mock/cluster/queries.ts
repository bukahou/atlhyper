/**
 * Cluster mock queries — returns data in the same shape as real Axios responses.
 *
 * Pages access data via `response.data.data` (list/detail) or
 * `response.data.events` (events).
 */

import type {
  PodItem, PodDetail,
  NodeItem, NodeDetail,
  DeploymentItem, DeploymentDetail,
  ServiceItem, ServiceDetail,
  NamespaceItem, NamespaceDetail,
  IngressItem, IngressDetail,
  EventLog,
  ConfigMapDTO, SecretDTO,
} from "@/types/cluster";
import type { StatefulSetListItem, DaemonSetListItem, StatefulSetDetail, DaemonSetDetail } from "@/api/workload";
import type {
  JobItem, CronJobItem, PVItem, PVCItem,
  NetworkPolicyItem, ResourceQuotaItem, LimitRangeItem, ServiceAccountItem,
  JobDetail, CronJobDetail, PVDetail, PVCDetail,
  NetworkPolicyDetail, ResourceQuotaDetail, LimitRangeDetail, ServiceAccountDetail,
} from "@/api/cluster-resources";

import {
  MOCK_PODS, MOCK_NODES, MOCK_NAMESPACES, MOCK_DEPLOYMENTS,
  MOCK_SERVICES, MOCK_INGRESSES, MOCK_EVENTS,
} from "./data";
import {
  MOCK_STATEFULSETS, MOCK_DAEMONSETS, MOCK_JOBS, MOCK_CRONJOBS,
  MOCK_PVS, MOCK_PVCS, MOCK_NETWORK_POLICIES, MOCK_RESOURCE_QUOTAS,
  MOCK_LIMIT_RANGES, MOCK_SERVICE_ACCOUNTS, MOCK_CONFIGMAPS, MOCK_SECRETS,
} from "./data-extra";

// ============================================================
// Response wrappers
// ============================================================

function wrapResponse<T>(data: T) {
  return { data: { message: "OK", data, total: Array.isArray(data) ? data.length : 1 } };
}

function wrapOverview<T>(data: T) {
  return { data: { data } };
}

// ============================================================
// Helper: namespace filter
// ============================================================

function filterByNamespace<T extends { namespace: string }>(
  items: T[],
  namespace?: string,
): T[] {
  if (!namespace) return items;
  return items.filter((i) => i.namespace === namespace);
}

// ============================================================
// Pod
// ============================================================

export function mockGetPodList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_PODS, params?.namespace);
  return wrapResponse(filtered);
}

export function mockGetPodOverview() {
  const cards = {
    running: MOCK_PODS.filter((p) => p.phase === "Running").length,
    pending: MOCK_PODS.filter((p) => p.phase === "Pending").length,
    failed: MOCK_PODS.filter((p) => p.phase === "Failed").length,
    unknown: MOCK_PODS.filter((p) => p.phase === "Unknown").length,
  };
  return wrapOverview({ cards, pods: MOCK_PODS });
}

export function mockGetPodDetail(name: string, namespace: string) {
  const pod = MOCK_PODS.find((p) => p.name === name && p.namespace === namespace);
  if (!pod) return wrapResponse(null);

  const detail: PodDetail = {
    name: pod.name,
    namespace: pod.namespace,
    controller: pod.deployment,
    phase: pod.phase,
    ready: pod.ready,
    restarts: pod.restarts,
    startTime: pod.startTime,
    age: pod.age,
    node: pod.node,
    podIP: `10.42.${Math.floor(Math.random() * 255)}.${Math.floor(Math.random() * 255)}`,
    qosClass: "Burstable",
    containers: [
      {
        name: "app",
        image: pod.deployment ? `${pod.deployment.replace("geass-", "geass/")}:1.4.2` : "unknown:latest",
        state: pod.phase === "Running" ? "running" : "waiting",
        restartCount: pod.restarts,
      },
    ],
  };
  return wrapResponse(detail);
}

// ============================================================
// Node
// ============================================================

export function mockGetNodeList() {
  return wrapResponse(MOCK_NODES);
}

export function mockGetNodeOverview() {
  const cards = {
    totalNodes: MOCK_NODES.length,
    readyNodes: MOCK_NODES.filter((n) => n.ready).length,
    totalCPU: MOCK_NODES.reduce((s, n) => s + n.cpuCores, 0),
    totalMemoryGiB: MOCK_NODES.reduce((s, n) => s + n.memoryGiB, 0),
  };
  return wrapOverview({ cards, rows: MOCK_NODES });
}

export function mockGetNodeDetail(name: string) {
  const node = MOCK_NODES.find((n) => n.name === name);
  if (!node) return wrapResponse(null);

  const detail: NodeDetail = {
    name: node.name,
    roles: node.schedulable ? ["worker"] : ["control-plane"],
    ready: node.ready,
    schedulable: node.schedulable,
    age: "11d",
    createdAt: "2026-02-10T02:55:00Z",
    hostname: node.name,
    internalIP: node.internalIP,
    osImage: node.osImage,
    os: "linux",
    architecture: node.architecture,
    kernel: node.architecture === "arm64" ? "6.1.0-rpi7-rpi-2712" : "6.8.0-51-generic",
    cri: "containerd://1.7.22",
    kubelet: "v1.31.4",
    kubeProxy: "v1.31.4",
    cpuCapacityCores: node.cpuCores,
    cpuAllocatableCores: node.cpuCores,
    memCapacityGiB: node.memoryGiB,
    memAllocatableGiB: Math.round(node.memoryGiB * 0.95 * 10) / 10,
    podsCapacity: 110,
    podsAllocatable: 110,
    cpuUsageCores: +(node.cpuCores * 0.35).toFixed(2),
    cpuUtilPct: 35,
    memUsageGiB: +(node.memoryGiB * 0.6).toFixed(1),
    memUtilPct: 60,
    podsUsed: node.schedulable ? 10 : 6,
    podsUtilPct: node.schedulable ? 9 : 5,
    conditions: [
      { type: "Ready", status: "True", reason: "KubeletReady", message: "kubelet is posting ready status" },
      { type: "MemoryPressure", status: "False", reason: "KubeletHasSufficientMemory" },
      { type: "DiskPressure", status: "False", reason: "KubeletHasNoDiskPressure" },
      { type: "PIDPressure", status: "False", reason: "KubeletHasSufficientPID" },
    ],
  };
  return wrapResponse(detail);
}

// ============================================================
// Deployment
// ============================================================

export function mockGetDeploymentList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_DEPLOYMENTS, params?.namespace);
  return wrapResponse(filtered);
}

export function mockGetDeploymentOverview() {
  const namespaces = new Set(MOCK_DEPLOYMENTS.map((d) => d.namespace));
  const totalReplicas = MOCK_DEPLOYMENTS.reduce((s, d) => {
    const parts = d.replicas.split("/");
    return s + parseInt(parts[1] || parts[0], 10);
  }, 0);
  const readyReplicas = MOCK_DEPLOYMENTS.reduce((s, d) => {
    const parts = d.replicas.split("/");
    return s + parseInt(parts[0], 10);
  }, 0);
  const cards = {
    totalDeployments: MOCK_DEPLOYMENTS.length,
    namespaces: namespaces.size,
    totalReplicas,
    readyReplicas,
  };
  return wrapOverview({ cards, rows: MOCK_DEPLOYMENTS });
}

export function mockGetDeploymentDetail(name: string, namespace: string) {
  const dep = MOCK_DEPLOYMENTS.find((d) => d.name === name && d.namespace === namespace);
  if (!dep) return wrapResponse(null);

  const parts = dep.replicas.split("/");
  const readyCount = parseInt(parts[0], 10);
  const desiredCount = parseInt(parts[1] || parts[0], 10);

  const detail: DeploymentDetail = {
    name: dep.name,
    namespace: dep.namespace,
    strategy: "RollingUpdate",
    replicas: desiredCount,
    updated: desiredCount,
    ready: readyCount,
    available: readyCount,
    createdAt: dep.createdAt,
    age: "2d",
    spec: {
      replicas: desiredCount,
      revisionHistoryLimit: 10,
      progressDeadlineSeconds: 600,
      strategyType: "RollingUpdate",
      maxUnavailable: "25%",
      maxSurge: "25%",
    },
    template: {
      labels: { app: dep.name },
      containers: [
        {
          name: dep.name,
          image: dep.image,
          ports: [{ containerPort: 8080, protocol: "TCP" }],
        },
      ],
    },
    status: {
      replicas: desiredCount,
      updatedReplicas: desiredCount,
      readyReplicas: readyCount,
      availableReplicas: readyCount,
    },
    conditions: [
      { type: "Available", status: "True", reason: "MinimumReplicasAvailable", message: "Deployment has minimum availability." },
      { type: "Progressing", status: "True", reason: "NewReplicaSetAvailable", message: `ReplicaSet "${dep.name}-xxx" has successfully progressed.` },
    ],
    labels: { app: dep.name },
  };
  return wrapResponse(detail);
}

// ============================================================
// Service
// ============================================================

export function mockGetServiceList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_SERVICES, params?.namespace);
  return wrapResponse(filtered);
}

export function mockGetServiceOverview() {
  const cards = {
    totalServices: MOCK_SERVICES.length,
    externalServices: MOCK_SERVICES.filter((s) => s.type === "LoadBalancer" || s.type === "NodePort").length,
    internalServices: MOCK_SERVICES.filter((s) => s.type === "ClusterIP" && s.clusterIP !== "None").length,
    headlessServices: MOCK_SERVICES.filter((s) => s.clusterIP === "None").length,
  };
  return wrapOverview({ cards, rows: MOCK_SERVICES });
}

export function mockGetServiceDetail(name: string, namespace: string) {
  const svc = MOCK_SERVICES.find(s => s.name === name && s.namespace === namespace);
  if (!svc) return wrapResponse(null);
  const detail: ServiceDetail = {
    name: svc.name,
    namespace: svc.namespace,
    type: svc.type,
    createdAt: svc.createdAt,
    age: "2d",
    sessionAffinity: "None",
    clusterIPs: [svc.clusterIP],
    externalIPs: svc.type === "LoadBalancer" ? ["203.0.113.10"] : [],
    loadBalancerIngress: svc.type === "LoadBalancer" ? ["203.0.113.10"] : [],
    externalTrafficPolicy: svc.type !== "ClusterIP" ? "Cluster" : undefined,
    internalTrafficPolicy: "Cluster",
    ipFamilies: ["IPv4"],
    ipFamilyPolicy: "SingleStack",
    ports: [{ protocol: "TCP", port: 80, targetPort: "8080" }],
    selector: { app: svc.name },
    backends: {
      ready: 2, notReady: 0, total: 2,
      endpoints: [
        { address: "10.42.1.15", ready: true, nodeName: "node-worker-01", targetRef: { kind: "Pod", name: `${svc.name}-xxx-abc` } },
        { address: "10.42.2.22", ready: true, nodeName: "node-worker-02", targetRef: { kind: "Pod", name: `${svc.name}-xxx-def` } },
      ],
    },
  };
  return wrapResponse(detail);
}

// ============================================================
// Namespace
// ============================================================

export function mockGetNamespaceList() {
  return wrapResponse(MOCK_NAMESPACES);
}

export function mockGetNamespaceOverview() {
  const cards = {
    totalNamespaces: MOCK_NAMESPACES.length,
    activeCount: MOCK_NAMESPACES.filter((n) => n.status === "Active").length,
    terminating: MOCK_NAMESPACES.filter((n) => n.status === "Terminating").length,
    totalPods: MOCK_NAMESPACES.reduce((s, n) => s + n.podCount, 0),
  };
  return wrapOverview({ cards, rows: MOCK_NAMESPACES });
}

export function mockGetNamespaceDetail(name: string) {
  const ns = MOCK_NAMESPACES.find(n => n.name === name);
  if (!ns) return wrapResponse(null);
  const detail: NamespaceDetail = {
    name: ns.name,
    phase: ns.status,
    createdAt: ns.createdAt,
    age: "2d",
    labels: { "kubernetes.io/metadata.name": ns.name },
    annotations: {},
    labelCount: ns.labelCount,
    annotationCount: ns.annotationCount,
    pods: ns.podCount,
    podsRunning: Math.max(0, ns.podCount - 1),
    podsPending: 0,
    podsFailed: 0,
    podsSucceeded: 1,
    deployments: 2,
    statefulSets: 0,
    daemonSets: 0,
    jobs: 1,
    cronJobs: 1,
    services: 3,
    ingresses: 2,
    configMaps: 2,
    secrets: 2,
    persistentVolumeClaims: 1,
    networkPolicies: 1,
    serviceAccounts: 1,
  };
  return wrapResponse(detail);
}

export function mockGetConfigMapList(namespace: string) {
  const filtered = MOCK_CONFIGMAPS.filter((c) => c.namespace === namespace);
  return wrapResponse(filtered);
}

export function mockGetSecretList(namespace: string) {
  const filtered = MOCK_SECRETS.filter((s) => s.namespace === namespace);
  return wrapResponse(filtered);
}

// ============================================================
// Ingress
// ============================================================

export function mockGetIngressList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_INGRESSES, params?.namespace);
  return wrapResponse(filtered);
}

export function mockGetIngressOverview() {
  const hosts = new Set(MOCK_INGRESSES.map((i) => i.host));
  const tlsCerts = new Set(
    MOCK_INGRESSES.filter((i) => i.tls).map((i) => i.host),
  );
  const cards = {
    totalIngresses: MOCK_INGRESSES.length,
    usedHosts: hosts.size,
    tlsCerts: tlsCerts.size,
    totalPaths: MOCK_INGRESSES.length,
  };
  return wrapOverview({ cards, rows: MOCK_INGRESSES });
}

export function mockGetIngressDetail(name: string, namespace: string) {
  const ing = MOCK_INGRESSES.find(i => i.name === name && i.namespace === namespace);
  if (!ing) return wrapResponse(null);
  const detail: IngressDetail = {
    name: ing.name,
    namespace: ing.namespace,
    class: "traefik",
    controller: "traefik.io/ingress-controller",
    hosts: [ing.host],
    tlsEnabled: ing.tls,
    loadBalancer: ["192.168.1.100"],
    createdAt: ing.createdAt,
    age: "2d",
    spec: {
      ingressClassName: "traefik",
      rules: [{
        host: ing.host,
        paths: [{
          path: ing.path,
          pathType: "Prefix",
          backend: { type: "Service", service: { name: ing.serviceName, portNumber: parseInt(ing.servicePort) || 80 } },
        }],
      }],
      tls: ing.tls ? [{ secretName: `${ing.name}-tls`, hosts: [ing.host] }] : [],
    },
    status: { loadBalancer: ["192.168.1.100"] },
    annotations: { "traefik.ingress.kubernetes.io/router.entrypoints": "websecure" },
  };
  return wrapResponse(detail);
}

// ============================================================
// Event
// ============================================================

export function mockGetEventList(params?: { namespace?: string; type?: string }) {
  let filtered = MOCK_EVENTS;
  if (params?.namespace) {
    filtered = filtered.filter((e) => e.namespace === params.namespace);
  }
  if (params?.type) {
    filtered = filtered.filter((e) => e.severity === params.type);
  }
  return { data: { events: filtered, total: filtered.length } };
}

export function mockGetEventOverview() {
  const kinds = new Set(MOCK_EVENTS.map((e) => e.kind));
  const categories = new Set(MOCK_EVENTS.map((e) => e.category));
  const cards = {
    totalAlerts: MOCK_EVENTS.filter((e) => e.severity === "Warning").length,
    totalEvents: MOCK_EVENTS.length,
    warning: MOCK_EVENTS.filter((e) => e.severity === "Warning").length,
    error: MOCK_EVENTS.filter((e) => e.severity === "Error").length,
    info: MOCK_EVENTS.filter((e) => e.severity === "Normal").length,
    kindsCount: kinds.size,
    categoriesCount: categories.size,
  };
  return wrapOverview({ cards, rows: MOCK_EVENTS });
}

// ============================================================
// StatefulSet & DaemonSet
// ============================================================

export function mockGetStatefulSetList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_STATEFULSETS, params?.namespace);
  return wrapResponse(filtered);
}

export function mockGetDaemonSetList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_DAEMONSETS, params?.namespace);
  return wrapResponse(filtered);
}

// ============================================================
// Job & CronJob
// ============================================================

export function mockGetJobList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_JOBS, params?.namespace);
  return wrapResponse(filtered);
}

export function mockGetCronJobList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_CRONJOBS, params?.namespace);
  return wrapResponse(filtered);
}

// ============================================================
// PV & PVC
// ============================================================

export function mockGetPVList() {
  return wrapResponse(MOCK_PVS);
}

export function mockGetPVCList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_PVCS, params?.namespace);
  return wrapResponse(filtered);
}

// ============================================================
// NetworkPolicy
// ============================================================

export function mockGetNetworkPolicyList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_NETWORK_POLICIES, params?.namespace);
  return wrapResponse(filtered);
}

// ============================================================
// ResourceQuota & LimitRange
// ============================================================

export function mockGetResourceQuotaList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_RESOURCE_QUOTAS, params?.namespace);
  return wrapResponse(filtered);
}

export function mockGetLimitRangeList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_LIMIT_RANGES, params?.namespace);
  return wrapResponse(filtered);
}

// ============================================================
// ServiceAccount
// ============================================================

export function mockGetServiceAccountList(params?: { namespace?: string }) {
  const filtered = filterByNamespace(MOCK_SERVICE_ACCOUNTS, params?.namespace);
  return wrapResponse(filtered);
}

// ============================================================
// Extra resource detail helpers (full Detail structure)
// ============================================================

export function mockGetStatefulSetDetail(name: string, namespace: string) {
  const sts = MOCK_STATEFULSETS.find(i => i.name === name && i.namespace === namespace);
  if (!sts) return wrapResponse(null);
  const detail: StatefulSetDetail = {
    name: sts.name,
    namespace: sts.namespace,
    replicas: sts.replicas,
    ready: sts.ready,
    current: sts.current,
    updated: sts.updated,
    available: sts.available,
    createdAt: sts.createdAt,
    age: sts.age,
    serviceName: sts.serviceName,
    selector: `app=${sts.name}`,
    spec: {
      podManagementPolicy: "OrderedReady",
      updateStrategy: { type: "RollingUpdate", partition: 0 },
      revisionHistoryLimit: 10,
      volumeClaimTemplates: [{ name: "data", accessModes: ["ReadWriteOnce"], storageClass: "nfs", storage: "10Gi" }],
    },
    template: {
      containers: [{
        name: sts.name,
        image: `${sts.name}:7.2`,
        ports: [{ containerPort: 6379, protocol: "TCP" }],
        requests: { cpu: "100m", memory: "128Mi" },
        limits: { cpu: "500m", memory: "512Mi" },
      }],
    },
    status: { currentRevision: `${sts.name}-5f4b8c9d7`, updateRevision: `${sts.name}-5f4b8c9d7` },
    conditions: [
      { type: "Available", status: "True", reason: "MinimumReplicasAvailable" },
    ],
    labels: { app: sts.name, "app.kubernetes.io/name": sts.name },
  };
  return wrapResponse(detail);
}

export function mockGetDaemonSetDetail(name: string, namespace: string) {
  const ds = MOCK_DAEMONSETS.find(i => i.name === name && i.namespace === namespace);
  if (!ds) return wrapResponse(null);
  const detail: DaemonSetDetail = {
    name: ds.name,
    namespace: ds.namespace,
    desired: ds.desired,
    current: ds.current,
    ready: ds.ready,
    available: ds.available,
    unavailable: 0,
    misscheduled: ds.misscheduled,
    updatedScheduled: ds.current,
    createdAt: ds.createdAt,
    age: ds.age,
    selector: `app=${ds.name}`,
    spec: {
      updateStrategy: { type: "RollingUpdate", maxUnavailable: "1" },
      revisionHistoryLimit: 10,
    },
    template: {
      containers: [{
        name: ds.name,
        image: `k8s.gcr.io/${ds.name}:v1.31.0`,
        ports: [{ containerPort: 10256, protocol: "TCP" }],
        requests: { cpu: "100m", memory: "64Mi" },
      }],
    },
    conditions: [
      { type: "Available", status: "True", reason: "MinimumReplicasAvailable" },
    ],
    labels: { "k8s-app": ds.name },
  };
  return wrapResponse(detail);
}

export function mockGetJobDetail(name: string, namespace: string) {
  const job = MOCK_JOBS.find(i => i.name === name && i.namespace === namespace);
  if (!job) return wrapResponse(null);
  const detail: JobDetail = {
    name: job.name,
    namespace: job.namespace,
    uid: `uid-${job.name}`,
    createdAt: job.createdAt,
    age: job.age,
    status: job.complete ? "Complete" : "Running",
    active: job.active,
    succeeded: job.succeeded,
    failed: job.failed,
    completions: 1,
    parallelism: 1,
    backoffLimit: 6,
    startTime: job.startTime,
    finishTime: job.finishTime,
    duration: "2m15s",
    template: {
      containers: [{
        name: job.name,
        image: `geass/${job.name.split("-")[0]}:latest`,
        command: ["/bin/sh", "-c"],
        args: ["echo 'Job running'"],
      }],
      serviceAccountName: "default",
    },
    conditions: job.complete ? [
      { type: "Complete", status: "True", reason: "JobComplete", lastTransitionTime: job.finishTime },
    ] : [],
    labels: { "batch.kubernetes.io/job-name": job.name },
  };
  return wrapResponse(detail);
}

export function mockGetCronJobDetail(name: string, namespace: string) {
  const cj = MOCK_CRONJOBS.find(i => i.name === name && i.namespace === namespace);
  if (!cj) return wrapResponse(null);
  const detail: CronJobDetail = {
    name: cj.name,
    namespace: cj.namespace,
    uid: `uid-${cj.name}`,
    createdAt: cj.createdAt,
    age: cj.age,
    schedule: cj.schedule,
    suspend: cj.suspend,
    concurrencyPolicy: "Forbid",
    activeJobs: cj.activeJobs,
    successfulJobsHistoryLimit: 3,
    failedJobsHistoryLimit: 1,
    lastScheduleTime: cj.lastScheduleTime,
    lastSuccessfulTime: cj.lastSuccessfulTime,
    lastScheduleAgo: "19h",
    lastSuccessAgo: "19h",
    template: {
      containers: [{
        name: cj.name,
        image: `geass/${cj.name}:latest`,
        command: ["/bin/sh", "-c"],
        args: ["echo 'CronJob running'"],
      }],
    },
    labels: { app: cj.name },
  };
  return wrapResponse(detail);
}

export function mockGetPVDetail(name: string) {
  const pv = MOCK_PVS.find(i => i.name === name);
  if (!pv) return wrapResponse(null);
  const detail: PVDetail = {
    name: pv.name,
    uid: `uid-${pv.name}`,
    capacity: pv.capacity,
    phase: pv.phase,
    storageClass: pv.storageClass,
    accessModes: pv.accessModes,
    reclaimPolicy: pv.reclaimPolicy,
    volumeSourceType: "NFS",
    claimRefName: pv.name.replace("pv", "pvc"),
    claimRefNamespace: "geass",
    createdAt: pv.createdAt,
    age: pv.age,
    labels: { "storage-tier": "standard" },
  };
  return wrapResponse(detail);
}

export function mockGetPVCDetail(name: string, namespace: string) {
  const pvc = MOCK_PVCS.find(i => i.name === name && i.namespace === namespace);
  if (!pvc) return wrapResponse(null);
  const detail: PVCDetail = {
    name: pvc.name,
    namespace: pvc.namespace,
    uid: `uid-${pvc.name}`,
    phase: pvc.phase,
    volumeName: pvc.volumeName,
    storageClass: pvc.storageClass,
    accessModes: pvc.accessModes,
    requestedCapacity: pvc.requestedCapacity,
    actualCapacity: pvc.actualCapacity,
    volumeMode: "Filesystem",
    createdAt: pvc.createdAt,
    age: pvc.age,
    labels: { app: "geass" },
  };
  return wrapResponse(detail);
}

export function mockGetNetworkPolicyDetail(name: string, namespace: string) {
  const np = MOCK_NETWORK_POLICIES.find(i => i.name === name && i.namespace === namespace);
  if (!np) return wrapResponse(null);
  const detail: NetworkPolicyDetail = {
    name: np.name,
    namespace: np.namespace,
    podSelector: np.podSelector,
    policyTypes: np.policyTypes,
    ingressRuleCount: np.ingressRuleCount,
    egressRuleCount: np.egressRuleCount,
    ingressRules: np.ingressRuleCount > 0 ? [{
      peers: [{ type: "namespaceSelector", selector: "kubernetes.io/metadata.name=traefik" }],
      ports: [{ protocol: "TCP", port: "8080" }],
    }] : [],
    egressRules: [],
    createdAt: np.createdAt,
    age: np.age,
    labels: { app: "geass" },
  };
  return wrapResponse(detail);
}

export function mockGetResourceQuotaDetail(name: string, namespace: string) {
  const rq = MOCK_RESOURCE_QUOTAS.find(i => i.name === name && i.namespace === namespace);
  if (!rq) return wrapResponse(null);
  const detail: ResourceQuotaDetail = {
    name: rq.name,
    namespace: rq.namespace,
    scopes: rq.scopes,
    hard: rq.hard,
    used: rq.used,
    createdAt: rq.createdAt,
    age: rq.age,
    labels: { app: "geass" },
  };
  return wrapResponse(detail);
}

export function mockGetLimitRangeDetail(name: string, namespace: string) {
  const lr = MOCK_LIMIT_RANGES.find(i => i.name === name && i.namespace === namespace);
  if (!lr) return wrapResponse(null);
  const detail: LimitRangeDetail = {
    name: lr.name,
    namespace: lr.namespace,
    items: lr.items,
    createdAt: lr.createdAt,
    age: lr.age,
    labels: { app: "geass" },
  };
  return wrapResponse(detail);
}

export function mockGetServiceAccountDetail(name: string, namespace: string) {
  const sa = MOCK_SERVICE_ACCOUNTS.find(i => i.name === name && i.namespace === namespace);
  if (!sa) return wrapResponse(null);
  const detail: ServiceAccountDetail = {
    name: sa.name,
    namespace: sa.namespace,
    secretsCount: sa.secretsCount,
    imagePullSecretsCount: sa.imagePullSecretsCount,
    automountServiceAccountToken: sa.automountServiceAccountToken,
    secretNames: sa.secretsCount > 0 ? [`${sa.name}-token-xxxxx`] : [],
    imagePullSecretNames: [],
    createdAt: sa.createdAt,
    age: sa.age,
    labels: sa.name === "default" ? {} : { app: sa.name },
  };
  return wrapResponse(detail);
}
