export type EntityType =
  | "pod"
  | "node"
  | "service"
  | "ingress"
  | "deployment"
  | "statefulset"
  | "daemonset"
  | "job"
  | "cronjob"
  | "namespace"
  | "pv"
  | "pvc"
  | "network-policy"
  | "resource-quota"
  | "limit-range"
  | "service-account";

export interface EntityDetailTarget {
  type: EntityType;
  name: string;
  namespace: string; // Node/PV/Namespace 传 ""
}
