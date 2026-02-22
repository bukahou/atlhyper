/**
 * K8s 集群资源 Mock 数据
 *
 * 集群: zgmf-x10a
 * 节点: raspi-nfs (control-plane), jegan-worker-01, jegan-worker-02
 * 应用: geass 命名空间下的 Spring Boot 微服务群
 */

import type {
  PodItem,
  NodeItem,
  NamespaceItem,
  DeploymentItem,
  ServiceItem,
  IngressItem,
  EventLog,
} from "@/types/cluster";

// ==================== Pods (~25) ====================

export const MOCK_PODS: PodItem[] = [
  // --- geass namespace (6 services, on workers) ---
  { name: "geass-gateway-7b8d4f6c9-x2k9p", namespace: "geass", deployment: "geass-gateway", ready: "1/1", phase: "Running", restarts: 0, cpuText: "45m", memoryText: "256Mi", startTime: "2026-02-19T08:30:00Z", node: "jegan-worker-01", age: "2d" },
  { name: "geass-auth-6d5c8e7f1-m4n7q", namespace: "geass", deployment: "geass-auth", ready: "1/1", phase: "Running", restarts: 0, cpuText: "32m", memoryText: "192Mi", startTime: "2026-02-19T08:31:00Z", node: "jegan-worker-02", age: "2d" },
  { name: "geass-media-5a9b3c2d8-r6t1w", namespace: "geass", deployment: "geass-media", ready: "1/1", phase: "Running", restarts: 0, cpuText: "68m", memoryText: "384Mi", startTime: "2026-02-19T08:32:00Z", node: "jegan-worker-01", age: "2d" },
  { name: "geass-favorites-8f2e1a7b4-h5j3v", namespace: "geass", deployment: "geass-favorites", ready: "1/1", phase: "Running", restarts: 0, cpuText: "18m", memoryText: "128Mi", startTime: "2026-02-19T08:33:00Z", node: "jegan-worker-02", age: "2d" },
  { name: "geass-history-3c7d9e6f2-p8s5y", namespace: "geass", deployment: "geass-history", ready: "1/1", phase: "Running", restarts: 2, cpuText: "25m", memoryText: "160Mi", startTime: "2026-02-20T14:10:00Z", node: "jegan-worker-01", age: "1d" },
  { name: "geass-user-9a1b4c8d5-k2m6n", namespace: "geass", deployment: "geass-user", ready: "1/1", phase: "Running", restarts: 0, cpuText: "22m", memoryText: "144Mi", startTime: "2026-02-19T08:35:00Z", node: "jegan-worker-02", age: "2d" },
  // --- kube-system (10 pods) ---
  { name: "coredns-5d78c9869-4fzxl", namespace: "kube-system", deployment: "coredns", ready: "1/1", phase: "Running", restarts: 0, cpuText: "8m", memoryText: "24Mi", startTime: "2026-02-10T03:00:00Z", node: "jegan-worker-01", age: "11d" },
  { name: "coredns-5d78c9869-b7nqr", namespace: "kube-system", deployment: "coredns", ready: "1/1", phase: "Running", restarts: 0, cpuText: "7m", memoryText: "22Mi", startTime: "2026-02-10T03:00:00Z", node: "jegan-worker-02", age: "11d" },
  { name: "etcd-raspi-nfs", namespace: "kube-system", deployment: "", ready: "1/1", phase: "Running", restarts: 0, cpuText: "35m", memoryText: "64Mi", startTime: "2026-02-10T02:55:00Z", node: "raspi-nfs", age: "11d" },
  { name: "kube-apiserver-raspi-nfs", namespace: "kube-system", deployment: "", ready: "1/1", phase: "Running", restarts: 0, cpuText: "82m", memoryText: "312Mi", startTime: "2026-02-10T02:55:00Z", node: "raspi-nfs", age: "11d" },
  { name: "kube-controller-manager-raspi-nfs", namespace: "kube-system", deployment: "", ready: "1/1", phase: "Running", restarts: 0, cpuText: "28m", memoryText: "56Mi", startTime: "2026-02-10T02:55:00Z", node: "raspi-nfs", age: "11d" },
  { name: "kube-scheduler-raspi-nfs", namespace: "kube-system", deployment: "", ready: "1/1", phase: "Running", restarts: 0, cpuText: "12m", memoryText: "32Mi", startTime: "2026-02-10T02:55:00Z", node: "raspi-nfs", age: "11d" },
  { name: "kube-proxy-r4f8k", namespace: "kube-system", deployment: "", ready: "1/1", phase: "Running", restarts: 0, cpuText: "2m", memoryText: "18Mi", startTime: "2026-02-10T03:00:00Z", node: "raspi-nfs", age: "11d" },
  { name: "kube-proxy-w9j2m", namespace: "kube-system", deployment: "", ready: "1/1", phase: "Running", restarts: 0, cpuText: "2m", memoryText: "18Mi", startTime: "2026-02-10T03:00:00Z", node: "jegan-worker-01", age: "11d" },
  { name: "kube-proxy-t5n1p", namespace: "kube-system", deployment: "", ready: "1/1", phase: "Running", restarts: 0, cpuText: "2m", memoryText: "18Mi", startTime: "2026-02-10T03:00:00Z", node: "jegan-worker-02", age: "11d" },
  // --- linkerd (3 pods) ---
  { name: "linkerd-destination-6b4a2c8d1-q7w3e", namespace: "linkerd", deployment: "linkerd-destination", ready: "2/2", phase: "Running", restarts: 0, cpuText: "15m", memoryText: "96Mi", startTime: "2026-02-12T10:00:00Z", node: "jegan-worker-01", age: "9d" },
  { name: "linkerd-identity-9f3e7b5a2-u8r4t", namespace: "linkerd", deployment: "linkerd-identity", ready: "2/2", phase: "Running", restarts: 0, cpuText: "10m", memoryText: "64Mi", startTime: "2026-02-12T10:00:00Z", node: "jegan-worker-02", age: "9d" },
  { name: "linkerd-proxy-injector-4d1c6e9a3-y5h2k", namespace: "linkerd", deployment: "linkerd-proxy-injector", ready: "2/2", phase: "Running", restarts: 0, cpuText: "8m", memoryText: "48Mi", startTime: "2026-02-12T10:00:00Z", node: "jegan-worker-01", age: "9d" },
  // --- traefik (1 pod) ---
  { name: "traefik-7c9d5e8f3-z6v1b", namespace: "traefik", deployment: "traefik", ready: "1/1", phase: "Running", restarts: 0, cpuText: "20m", memoryText: "80Mi", startTime: "2026-02-11T06:00:00Z", node: "jegan-worker-02", age: "10d" },
  // --- Pending pod (visual variety) ---
  { name: "geass-media-5a9b3c2d8-new7f", namespace: "geass", deployment: "geass-media", ready: "0/1", phase: "Pending", restarts: 0, cpuText: "0m", memoryText: "0Mi", startTime: "2026-02-21T09:15:00Z", node: "", age: "2h" },
  // --- High restarts pod (visual variety) ---
  { name: "linkerd-viz-tap-8e2f4a6b1-c3d9g", namespace: "linkerd-viz", deployment: "linkerd-viz-tap", ready: "1/2", phase: "Running", restarts: 47, cpuText: "52m", memoryText: "210Mi", startTime: "2026-02-14T12:00:00Z", node: "jegan-worker-01", age: "7d" },
];

// ==================== Nodes (3) ====================

export const MOCK_NODES: NodeItem[] = [
  { name: "raspi-nfs", ready: true, internalIP: "192.168.1.100", osImage: "Debian GNU/Linux 12 (bookworm)", architecture: "arm64", cpuCores: 4, memoryGiB: 8, schedulable: false },
  { name: "jegan-worker-01", ready: true, internalIP: "192.168.1.101", osImage: "Ubuntu 24.04 LTS", architecture: "amd64", cpuCores: 8, memoryGiB: 32, schedulable: true },
  { name: "jegan-worker-02", ready: true, internalIP: "192.168.1.102", osImage: "Ubuntu 24.04 LTS", architecture: "amd64", cpuCores: 8, memoryGiB: 32, schedulable: true },
];

// ==================== Namespaces (6) ====================

export const MOCK_NAMESPACES: NamespaceItem[] = [
  { name: "default", status: "Active", podCount: 0, labelCount: 1, annotationCount: 0, createdAt: "2026-01-15T00:00:00Z" },
  { name: "kube-system", status: "Active", podCount: 10, labelCount: 1, annotationCount: 0, createdAt: "2026-01-15T00:00:00Z" },
  { name: "geass", status: "Active", podCount: 7, labelCount: 3, annotationCount: 1, createdAt: "2026-02-01T09:00:00Z" },
  { name: "linkerd", status: "Active", podCount: 3, labelCount: 5, annotationCount: 2, createdAt: "2026-02-05T10:00:00Z" },
  { name: "linkerd-viz", status: "Active", podCount: 1, labelCount: 4, annotationCount: 1, createdAt: "2026-02-05T10:30:00Z" },
  { name: "traefik", status: "Active", podCount: 1, labelCount: 2, annotationCount: 0, createdAt: "2026-02-03T08:00:00Z" },
];

// ==================== Deployments (~12) ====================

export const MOCK_DEPLOYMENTS: DeploymentItem[] = [
  // geass services
  { name: "geass-gateway", namespace: "geass", image: "geass/gateway:1.4.2", replicas: "1/1", labelCount: 4, annoCount: 2, createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-auth", namespace: "geass", image: "geass/auth:1.4.2", replicas: "1/1", labelCount: 4, annoCount: 2, createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-media", namespace: "geass", image: "geass/media:1.4.2", replicas: "2/2", labelCount: 4, annoCount: 2, createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-favorites", namespace: "geass", image: "geass/favorites:1.4.2", replicas: "1/1", labelCount: 4, annoCount: 2, createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-history", namespace: "geass", image: "geass/history:1.4.2", replicas: "1/1", labelCount: 4, annoCount: 2, createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-user", namespace: "geass", image: "geass/user:1.4.2", replicas: "1/1", labelCount: 4, annoCount: 2, createdAt: "2026-02-19T08:30:00Z" },
  // kube-system
  { name: "coredns", namespace: "kube-system", image: "registry.k8s.io/coredns/coredns:v1.11.3", replicas: "2/2", labelCount: 2, annoCount: 0, createdAt: "2026-02-10T03:00:00Z" },
  // linkerd
  { name: "linkerd-destination", namespace: "linkerd", image: "cr.l5d.io/linkerd/controller:stable-2.14.10", replicas: "1/1", labelCount: 6, annoCount: 3, createdAt: "2026-02-12T10:00:00Z" },
  { name: "linkerd-identity", namespace: "linkerd", image: "cr.l5d.io/linkerd/controller:stable-2.14.10", replicas: "1/1", labelCount: 6, annoCount: 3, createdAt: "2026-02-12T10:00:00Z" },
  { name: "linkerd-proxy-injector", namespace: "linkerd", image: "cr.l5d.io/linkerd/controller:stable-2.14.10", replicas: "1/1", labelCount: 6, annoCount: 3, createdAt: "2026-02-12T10:00:00Z" },
  // linkerd-viz
  { name: "linkerd-viz-tap", namespace: "linkerd-viz", image: "cr.l5d.io/linkerd/tap:stable-2.14.10", replicas: "1/1", labelCount: 5, annoCount: 2, createdAt: "2026-02-14T12:00:00Z" },
  // traefik
  { name: "traefik", namespace: "traefik", image: "traefik:v3.2.3", replicas: "1/1", labelCount: 3, annoCount: 1, createdAt: "2026-02-11T06:00:00Z" },
];

// ==================== Services (~12) ====================

export const MOCK_SERVICES: ServiceItem[] = [
  // default
  { name: "kubernetes", namespace: "default", type: "ClusterIP", clusterIP: "10.96.0.1", ports: "443", protocol: "TCP", selector: "", createdAt: "2026-01-15T00:00:00Z" },
  // kube-system
  { name: "kube-dns", namespace: "kube-system", type: "ClusterIP", clusterIP: "10.96.0.10", ports: "53,9153", protocol: "UDP,TCP", selector: "k8s-app=kube-dns", createdAt: "2026-02-10T03:00:00Z" },
  // geass services
  { name: "geass-gateway", namespace: "geass", type: "ClusterIP", clusterIP: "10.96.12.10", ports: "8080", protocol: "TCP", selector: "app=geass-gateway", createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-auth", namespace: "geass", type: "ClusterIP", clusterIP: "10.96.12.11", ports: "8080", protocol: "TCP", selector: "app=geass-auth", createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-media", namespace: "geass", type: "ClusterIP", clusterIP: "10.96.12.12", ports: "8080", protocol: "TCP", selector: "app=geass-media", createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-favorites", namespace: "geass", type: "ClusterIP", clusterIP: "10.96.12.13", ports: "8080", protocol: "TCP", selector: "app=geass-favorites", createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-history", namespace: "geass", type: "ClusterIP", clusterIP: "10.96.12.14", ports: "8080", protocol: "TCP", selector: "app=geass-history", createdAt: "2026-02-19T08:30:00Z" },
  { name: "geass-user", namespace: "geass", type: "ClusterIP", clusterIP: "10.96.12.15", ports: "8080", protocol: "TCP", selector: "app=geass-user", createdAt: "2026-02-19T08:30:00Z" },
  // traefik (LoadBalancer)
  { name: "traefik", namespace: "traefik", type: "LoadBalancer", clusterIP: "10.96.20.1", ports: "80,443", protocol: "TCP", selector: "app.kubernetes.io/name=traefik", createdAt: "2026-02-11T06:00:00Z" },
  // linkerd
  { name: "linkerd-dst", namespace: "linkerd", type: "ClusterIP", clusterIP: "10.96.30.1", ports: "8086", protocol: "TCP", selector: "linkerd.io/control-plane-component=destination", createdAt: "2026-02-12T10:00:00Z" },
  { name: "linkerd-identity", namespace: "linkerd", type: "ClusterIP", clusterIP: "10.96.30.2", ports: "8080", protocol: "TCP", selector: "linkerd.io/control-plane-component=identity", createdAt: "2026-02-12T10:00:00Z" },
  { name: "linkerd-proxy-injector", namespace: "linkerd", type: "ClusterIP", clusterIP: "10.96.30.3", ports: "443", protocol: "TCP", selector: "linkerd.io/control-plane-component=proxy-injector", createdAt: "2026-02-12T10:00:00Z" },
];

// ==================== Ingresses (8 rows, host x path expanded) ====================

export const MOCK_INGRESSES: IngressItem[] = [
  { name: "geass-ingress", namespace: "geass", host: "geass.example.com", path: "/api/gateway", serviceName: "geass-gateway", servicePort: "8080", tls: true, createdAt: "2026-02-19T09:00:00Z" },
  { name: "geass-ingress", namespace: "geass", host: "geass.example.com", path: "/api/auth", serviceName: "geass-auth", servicePort: "8080", tls: true, createdAt: "2026-02-19T09:00:00Z" },
  { name: "geass-ingress", namespace: "geass", host: "geass.example.com", path: "/api/media", serviceName: "geass-media", servicePort: "8080", tls: true, createdAt: "2026-02-19T09:00:00Z" },
  { name: "geass-ingress", namespace: "geass", host: "geass.example.com", path: "/api/favorites", serviceName: "geass-favorites", servicePort: "8080", tls: true, createdAt: "2026-02-19T09:00:00Z" },
  { name: "geass-ingress", namespace: "geass", host: "geass.example.com", path: "/api/history", serviceName: "geass-history", servicePort: "8080", tls: true, createdAt: "2026-02-19T09:00:00Z" },
  { name: "geass-ingress", namespace: "geass", host: "geass.example.com", path: "/api/user", serviceName: "geass-user", servicePort: "8080", tls: true, createdAt: "2026-02-19T09:00:00Z" },
  { name: "traefik-dashboard", namespace: "traefik", host: "traefik.example.com", path: "/dashboard", serviceName: "traefik", servicePort: "8080", tls: false, createdAt: "2026-02-11T06:30:00Z" },
  { name: "linkerd-viz", namespace: "linkerd-viz", host: "linkerd.example.com", path: "/", serviceName: "linkerd-viz-web", servicePort: "8084", tls: true, createdAt: "2026-02-14T12:30:00Z" },
];

// ==================== Events (~15) ====================

const CID = "zgmf-x10a";

export const MOCK_EVENTS: EventLog[] = [
  // Normal: Pod scheduled & started
  { clusterId: CID, category: "pod", eventTime: "2026-02-21T09:15:00Z", kind: "Pod", message: "Successfully assigned geass/geass-media-5a9b3c2d8-new7f to jegan-worker-01", name: "geass-media-5a9b3c2d8-new7f", namespace: "geass", node: "jegan-worker-01", reason: "Scheduled", severity: "Normal", time: "2026-02-21T09:15:00Z", source: "default-scheduler", count: 1, firstTimestamp: "2026-02-21T09:15:00Z", lastTimestamp: "2026-02-21T09:15:00Z" },
  { clusterId: CID, category: "pod", eventTime: "2026-02-21T09:15:05Z", kind: "Pod", message: "Pulling image \"geass/media:1.4.2\"", name: "geass-media-5a9b3c2d8-new7f", namespace: "geass", node: "jegan-worker-01", reason: "Pulling", severity: "Normal", time: "2026-02-21T09:15:05Z", source: "kubelet", count: 1, firstTimestamp: "2026-02-21T09:15:05Z", lastTimestamp: "2026-02-21T09:15:05Z" },
  { clusterId: CID, category: "pod", eventTime: "2026-02-19T08:30:10Z", kind: "Pod", message: "Started container geass-gateway", name: "geass-gateway-7b8d4f6c9-x2k9p", namespace: "geass", node: "jegan-worker-01", reason: "Started", severity: "Normal", time: "2026-02-19T08:30:10Z", source: "kubelet", count: 1, firstTimestamp: "2026-02-19T08:30:10Z", lastTimestamp: "2026-02-19T08:30:10Z" },
  { clusterId: CID, category: "pod", eventTime: "2026-02-19T08:31:10Z", kind: "Pod", message: "Started container geass-auth", name: "geass-auth-6d5c8e7f1-m4n7q", namespace: "geass", node: "jegan-worker-02", reason: "Started", severity: "Normal", time: "2026-02-19T08:31:10Z", source: "kubelet", count: 1, firstTimestamp: "2026-02-19T08:31:10Z", lastTimestamp: "2026-02-19T08:31:10Z" },
  { clusterId: CID, category: "pod", eventTime: "2026-02-19T08:32:10Z", kind: "Pod", message: "Successfully pulled image \"geass/media:1.4.2\" in 3.2s", name: "geass-media-5a9b3c2d8-r6t1w", namespace: "geass", node: "jegan-worker-01", reason: "Pulled", severity: "Normal", time: "2026-02-19T08:32:10Z", source: "kubelet", count: 1, firstTimestamp: "2026-02-19T08:32:10Z", lastTimestamp: "2026-02-19T08:32:10Z" },
  // Warning: OOMKilled (Critical)
  { clusterId: CID, category: "pod", eventTime: "2026-02-20T14:08:00Z", kind: "Pod", message: "Container geass-history terminated (OOMKilled) with exit code 137", name: "geass-history-3c7d9e6f2-p8s5y", namespace: "geass", node: "jegan-worker-01", reason: "OOMKilling", severity: "Warning", time: "2026-02-20T14:08:00Z", source: "kubelet", count: 3, firstTimestamp: "2026-02-20T12:00:00Z", lastTimestamp: "2026-02-20T14:08:00Z" },
  { clusterId: CID, category: "pod", eventTime: "2026-02-20T14:10:00Z", kind: "Pod", message: "Back-off restarting failed container geass-history in pod geass-history-3c7d9e6f2-p8s5y", name: "geass-history-3c7d9e6f2-p8s5y", namespace: "geass", node: "jegan-worker-01", reason: "BackOff", severity: "Warning", time: "2026-02-20T14:10:00Z", source: "kubelet", count: 5, firstTimestamp: "2026-02-20T12:05:00Z", lastTimestamp: "2026-02-20T14:10:00Z" },
  // Warning: FailedScheduling
  { clusterId: CID, category: "pod", eventTime: "2026-02-21T09:15:00Z", kind: "Pod", message: "0/3 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane:}, 2 Insufficient memory.", name: "geass-media-5a9b3c2d8-new7f", namespace: "geass", node: "", reason: "FailedScheduling", severity: "Warning", time: "2026-02-21T09:15:00Z", source: "default-scheduler", count: 4, firstTimestamp: "2026-02-21T09:15:00Z", lastTimestamp: "2026-02-21T09:18:00Z" },
  // Normal: Node events
  { clusterId: CID, category: "node", eventTime: "2026-02-10T03:00:00Z", kind: "Node", message: "Node jegan-worker-01 status is now: NodeReady", name: "jegan-worker-01", namespace: "", node: "jegan-worker-01", reason: "NodeReady", severity: "Normal", time: "2026-02-10T03:00:00Z", source: "kubelet", count: 1, firstTimestamp: "2026-02-10T03:00:00Z", lastTimestamp: "2026-02-10T03:00:00Z" },
  { clusterId: CID, category: "node", eventTime: "2026-02-10T03:00:00Z", kind: "Node", message: "Node jegan-worker-02 status is now: NodeReady", name: "jegan-worker-02", namespace: "", node: "jegan-worker-02", reason: "NodeReady", severity: "Normal", time: "2026-02-10T03:00:00Z", source: "kubelet", count: 1, firstTimestamp: "2026-02-10T03:00:00Z", lastTimestamp: "2026-02-10T03:00:00Z" },
  // Warning: linkerd-viz high restarts
  { clusterId: CID, category: "pod", eventTime: "2026-02-21T07:45:00Z", kind: "Pod", message: "Back-off restarting failed container tap in pod linkerd-viz-tap-8e2f4a6b1-c3d9g", name: "linkerd-viz-tap-8e2f4a6b1-c3d9g", namespace: "linkerd-viz", node: "jegan-worker-01", reason: "BackOff", severity: "Warning", time: "2026-02-21T07:45:00Z", source: "kubelet", count: 47, firstTimestamp: "2026-02-14T12:30:00Z", lastTimestamp: "2026-02-21T07:45:00Z" },
  { clusterId: CID, category: "pod", eventTime: "2026-02-21T06:30:00Z", kind: "Pod", message: "Liveness probe failed: HTTP probe failed with statuscode: 503", name: "linkerd-viz-tap-8e2f4a6b1-c3d9g", namespace: "linkerd-viz", node: "jegan-worker-01", reason: "Unhealthy", severity: "Warning", time: "2026-02-21T06:30:00Z", source: "kubelet", count: 120, firstTimestamp: "2026-02-14T13:00:00Z", lastTimestamp: "2026-02-21T06:30:00Z" },
  // Normal: Deployment scaling
  { clusterId: CID, category: "deployment", eventTime: "2026-02-19T08:30:00Z", kind: "Deployment", message: "Scaled up replica set geass-gateway-7b8d4f6c9 to 1", name: "geass-gateway", namespace: "geass", node: "", reason: "ScalingReplicaSet", severity: "Normal", time: "2026-02-19T08:30:00Z", source: "deployment-controller", count: 1, firstTimestamp: "2026-02-19T08:30:00Z", lastTimestamp: "2026-02-19T08:30:00Z" },
  { clusterId: CID, category: "deployment", eventTime: "2026-02-19T08:30:00Z", kind: "Deployment", message: "Scaled up replica set geass-media-5a9b3c2d8 to 2", name: "geass-media", namespace: "geass", node: "", reason: "ScalingReplicaSet", severity: "Normal", time: "2026-02-19T08:30:00Z", source: "deployment-controller", count: 1, firstTimestamp: "2026-02-19T08:30:00Z", lastTimestamp: "2026-02-19T08:30:00Z" },
  // Warning: Image pull backoff
  { clusterId: CID, category: "pod", eventTime: "2026-02-21T09:18:00Z", kind: "Pod", message: "Back-off pulling image \"geass/media:1.4.2\"", name: "geass-media-5a9b3c2d8-new7f", namespace: "geass", node: "jegan-worker-01", reason: "ImagePullBackOff", severity: "Warning", time: "2026-02-21T09:18:00Z", source: "kubelet", count: 3, firstTimestamp: "2026-02-21T09:16:00Z", lastTimestamp: "2026-02-21T09:18:00Z" },
  // --- 新增事件：覆盖更多场景 ---
  // ReplicaSet events
  { clusterId: CID, category: "replicaset", eventTime: "2026-02-19T08:30:05Z", kind: "ReplicaSet", message: "Created pod: geass-gateway-7b8d4f6c9-x2k9p", name: "geass-gateway-7b8d4f6c9", namespace: "geass", node: "", reason: "SuccessfulCreate", severity: "Normal", time: "2026-02-19T08:30:05Z", source: "replicaset-controller", count: 1, firstTimestamp: "2026-02-19T08:30:05Z", lastTimestamp: "2026-02-19T08:30:05Z" },
  { clusterId: CID, category: "replicaset", eventTime: "2026-02-19T08:30:05Z", kind: "ReplicaSet", message: "Created pod: geass-media-5a9b3c2d8-r6t1w", name: "geass-media-5a9b3c2d8", namespace: "geass", node: "", reason: "SuccessfulCreate", severity: "Normal", time: "2026-02-19T08:30:05Z", source: "replicaset-controller", count: 2, firstTimestamp: "2026-02-19T08:30:05Z", lastTimestamp: "2026-02-19T08:30:06Z" },
  // StatefulSet event
  { clusterId: CID, category: "statefulset", eventTime: "2026-02-15T10:00:00Z", kind: "StatefulSet", message: "create Pod geass-redis-0 in StatefulSet geass-redis successful", name: "geass-redis", namespace: "geass", node: "", reason: "SuccessfulCreate", severity: "Normal", time: "2026-02-15T10:00:00Z", source: "statefulset-controller", count: 1, firstTimestamp: "2026-02-15T10:00:00Z", lastTimestamp: "2026-02-15T10:00:00Z" },
  // DaemonSet event
  { clusterId: CID, category: "daemonset", eventTime: "2026-02-10T03:01:00Z", kind: "DaemonSet", message: "Created pod: kube-proxy-r4f8k", name: "kube-proxy", namespace: "kube-system", node: "", reason: "SuccessfulCreate", severity: "Normal", time: "2026-02-10T03:01:00Z", source: "daemonset-controller", count: 3, firstTimestamp: "2026-02-10T03:01:00Z", lastTimestamp: "2026-02-10T03:01:05Z" },
  // Job events
  { clusterId: CID, category: "job", eventTime: "2026-02-20T02:00:00Z", kind: "Job", message: "Created pod: geass-db-migrate-28x7k", name: "geass-db-migrate", namespace: "geass", node: "", reason: "SuccessfulCreate", severity: "Normal", time: "2026-02-20T02:00:00Z", source: "job-controller", count: 1, firstTimestamp: "2026-02-20T02:00:00Z", lastTimestamp: "2026-02-20T02:00:00Z" },
  { clusterId: CID, category: "job", eventTime: "2026-02-20T02:05:00Z", kind: "Job", message: "Job completed", name: "geass-db-migrate", namespace: "geass", node: "", reason: "Completed", severity: "Normal", time: "2026-02-20T02:05:00Z", source: "job-controller", count: 1, firstTimestamp: "2026-02-20T02:05:00Z", lastTimestamp: "2026-02-20T02:05:00Z" },
  // CronJob event
  { clusterId: CID, category: "cronjob", eventTime: "2026-02-21T00:00:00Z", kind: "CronJob", message: "Created job geass-backup-28481280", name: "geass-backup", namespace: "geass", node: "", reason: "SuccessfulCreate", severity: "Normal", time: "2026-02-21T00:00:00Z", source: "cronjob-controller", count: 1, firstTimestamp: "2026-02-21T00:00:00Z", lastTimestamp: "2026-02-21T00:00:00Z" },
  // PVC events
  { clusterId: CID, category: "pvc", eventTime: "2026-02-15T10:00:10Z", kind: "PersistentVolumeClaim", message: "Successfully provisioned volume pvc-abc123 using nfs-provisioner", name: "data-geass-redis-0", namespace: "geass", node: "", reason: "ProvisioningSucceeded", severity: "Normal", time: "2026-02-15T10:00:10Z", source: "nfs-provisioner", count: 1, firstTimestamp: "2026-02-15T10:00:10Z", lastTimestamp: "2026-02-15T10:00:10Z" },
  { clusterId: CID, category: "pvc", eventTime: "2026-02-21T08:00:00Z", kind: "PersistentVolumeClaim", message: "Failed to bind: no persistent volumes available for this claim", name: "data-geass-cache-0", namespace: "geass", node: "", reason: "FailedBinding", severity: "Warning", time: "2026-02-21T08:00:00Z", source: "persistentvolume-controller", count: 12, firstTimestamp: "2026-02-21T06:00:00Z", lastTimestamp: "2026-02-21T08:00:00Z" },
  // Node events: NotReady & Rebooted
  { clusterId: CID, category: "node", eventTime: "2026-02-18T03:45:00Z", kind: "Node", message: "Node raspi-nfs status is now: NodeNotReady", name: "raspi-nfs", namespace: "", node: "raspi-nfs", reason: "NodeNotReady", severity: "Warning", time: "2026-02-18T03:45:00Z", source: "node-controller", count: 1, firstTimestamp: "2026-02-18T03:45:00Z", lastTimestamp: "2026-02-18T03:45:00Z" },
  { clusterId: CID, category: "node", eventTime: "2026-02-18T03:50:00Z", kind: "Node", message: "Node raspi-nfs has been rebooted, boot id: a1b2c3d4", name: "raspi-nfs", namespace: "", node: "raspi-nfs", reason: "Rebooted", severity: "Warning", time: "2026-02-18T03:50:00Z", source: "kubelet", count: 1, firstTimestamp: "2026-02-18T03:50:00Z", lastTimestamp: "2026-02-18T03:50:00Z" },
  // CrashLoopBackOff (Critical)
  { clusterId: CID, category: "pod", eventTime: "2026-02-21T05:00:00Z", kind: "Pod", message: "Back-off restarting failed container app in pod geass-history-3c7d9e6f2-p8s5y", name: "geass-history-3c7d9e6f2-p8s5y", namespace: "geass", node: "jegan-worker-01", reason: "CrashLoopBackOff", severity: "Warning", time: "2026-02-21T05:00:00Z", source: "kubelet", count: 8, firstTimestamp: "2026-02-20T14:10:00Z", lastTimestamp: "2026-02-21T05:00:00Z" },
  // FailedMount
  { clusterId: CID, category: "pod", eventTime: "2026-02-21T08:30:00Z", kind: "Pod", message: "Unable to attach or mount volumes: timed out waiting for the condition", name: "geass-media-5a9b3c2d8-new7f", namespace: "geass", node: "jegan-worker-01", reason: "FailedMount", severity: "Warning", time: "2026-02-21T08:30:00Z", source: "kubelet", count: 2, firstTimestamp: "2026-02-21T08:25:00Z", lastTimestamp: "2026-02-21T08:30:00Z" },
  // Service event
  { clusterId: CID, category: "service", eventTime: "2026-02-11T06:00:05Z", kind: "Service", message: "Ensuring load balancer", name: "traefik", namespace: "traefik", node: "", reason: "EnsuringLoadBalancer", severity: "Normal", time: "2026-02-11T06:00:05Z", source: "service-controller", count: 1, firstTimestamp: "2026-02-11T06:00:05Z", lastTimestamp: "2026-02-11T06:00:05Z" },
  { clusterId: CID, category: "service", eventTime: "2026-02-11T06:00:10Z", kind: "Service", message: "Ensured load balancer", name: "traefik", namespace: "traefik", node: "", reason: "EnsuredLoadBalancer", severity: "Normal", time: "2026-02-11T06:00:10Z", source: "service-controller", count: 1, firstTimestamp: "2026-02-11T06:00:10Z", lastTimestamp: "2026-02-11T06:00:10Z" },
];
