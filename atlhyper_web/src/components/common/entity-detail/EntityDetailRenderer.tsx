"use client";

import type { EntityDetailTarget } from "@/types/entity-detail";
import { PodDetailModal } from "@/components/pod/PodDetailModal";
import { NodeDetailModal } from "@/components/node/NodeDetailModal";
import { ServiceDetailModal } from "@/components/service/ServiceDetailModal";
import { IngressDetailModal } from "@/components/ingress/IngressDetailModal";
import { DeploymentDetailModal } from "@/components/deployment/DeploymentDetailModal";
import { StatefulSetDetailModal } from "@/components/statefulset/StatefulSetDetailModal";
import { DaemonSetDetailModal } from "@/components/daemonset/DaemonSetDetailModal";
import { JobDetailModal } from "@/components/job/JobDetailModal";
import { CronJobDetailModal } from "@/components/cronjob/CronJobDetailModal";
import { NamespaceDetailModal } from "@/components/namespace/NamespaceDetailModal";
import { PVDetailModal } from "@/components/pv/PVDetailModal";
import { PVCDetailModal } from "@/components/pvc/PVCDetailModal";
import { NetworkPolicyDetailModal } from "@/components/network-policy/NetworkPolicyDetailModal";
import { ResourceQuotaDetailModal } from "@/components/resource-quota/ResourceQuotaDetailModal";
import { LimitRangeDetailModal } from "@/components/limit-range/LimitRangeDetailModal";
import { ServiceAccountDetailModal } from "@/components/service-account/ServiceAccountDetailModal";

interface Props {
  target: EntityDetailTarget | null;
  onClose: () => void;
}

const noop = () => {};

export function EntityDetailRenderer({ target, onClose }: Props) {
  if (!target) return null;

  const { type, name, namespace } = target;
  const isOpen = true;

  switch (type) {
    case "pod":
      return (
        <PodDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          podName={name}
          onViewLogs={noop}
        />
      );
    case "node":
      return (
        <NodeDetailModal
          isOpen={isOpen}
          onClose={onClose}
          nodeName={name}
        />
      );
    case "service":
      return (
        <ServiceDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          serviceName={name}
        />
      );
    case "ingress":
      return (
        <IngressDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          ingressName={name}
        />
      );
    case "deployment":
      return (
        <DeploymentDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          deploymentName={name}
        />
      );
    case "statefulset":
      return (
        <StatefulSetDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    case "daemonset":
      return (
        <DaemonSetDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    case "job":
      return (
        <JobDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    case "cronjob":
      return (
        <CronJobDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    case "namespace":
      return (
        <NamespaceDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespaceName={name}
        />
      );
    case "pv":
      return (
        <PVDetailModal
          isOpen={isOpen}
          onClose={onClose}
          name={name}
        />
      );
    case "pvc":
      return (
        <PVCDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    case "network-policy":
      return (
        <NetworkPolicyDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    case "resource-quota":
      return (
        <ResourceQuotaDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    case "limit-range":
      return (
        <LimitRangeDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    case "service-account":
      return (
        <ServiceAccountDetailModal
          isOpen={isOpen}
          onClose={onClose}
          namespace={namespace}
          name={name}
        />
      );
    default:
      return null;
  }
}
