"use client";

import type { ServiceHealth } from "@/types/model/observe";
import type { ObserveLandingTranslations } from "@/types/i18n";
import { DetailCard, KV, NoData } from "./SectionDetailParts";

export function K8sDetail({ service, tl }: { service: ServiceHealth; tl: ObserveLandingTranslations }) {
  const { deployment, pods, k8sService, ingresses } = service;

  return (
    <div className="space-y-4">
      {/* Deployment */}
      <DetailCard title={tl.deploymentSection}>
        {deployment ? (
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <KV label={tl.replicas} value={deployment.replicas} />
            <KV label={tl.strategy} value={deployment.strategy} />
            <KV label={tl.age} value={deployment.age} />
            <KV label={tl.image} value={deployment.image} mono />
          </div>
        ) : <NoData />}
      </DetailCard>

      {/* K8s Service */}
      {k8sService && (
        <DetailCard title="Service">
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
            <KV label={tl.serviceType} value={k8sService.type} />
            <KV label={tl.clusterIP} value={k8sService.clusterIP} mono />
            <KV label={tl.ports} value={k8sService.ports} mono />
          </div>
        </DetailCard>
      )}

      {/* Pods */}
      <DetailCard title={tl.pods}>
        {pods && pods.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-muted text-left border-b border-[var(--border-color)]">
                  <th className="py-1.5 pr-3 font-medium">{tl.serviceHealth}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.phase}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.ready}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.restarts}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.node}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.age}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.cpu}</th>
                  <th className="py-1.5 font-medium">{tl.memory}</th>
                </tr>
              </thead>
              <tbody>
                {pods.map((pod) => (
                  <tr key={pod.name} className="border-b border-[var(--border-color)] last:border-b-0">
                    <td className="py-1.5 pr-3 text-default font-mono truncate max-w-[200px]">{pod.name}</td>
                    <td className="py-1.5 pr-3">
                      <span className={pod.phase === "Running" ? "text-green-500" : pod.phase === "Pending" ? "text-yellow-500" : "text-red-500"}>
                        {pod.phase}
                      </span>
                    </td>
                    <td className="py-1.5 pr-3 text-default">{pod.ready}</td>
                    <td className="py-1.5 pr-3">
                      <span className={pod.restarts > 0 ? "text-yellow-500" : "text-default"}>{pod.restarts}</span>
                    </td>
                    <td className="py-1.5 pr-3 text-muted">{pod.nodeName}</td>
                    <td className="py-1.5 pr-3 text-muted">{pod.age}</td>
                    <td className="py-1.5 pr-3 text-default font-mono">{pod.cpuUsage}</td>
                    <td className="py-1.5 text-default font-mono">{pod.memoryUsage}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : <NoData />}
      </DetailCard>

      {/* Ingress */}
      <DetailCard title={tl.ingressSection}>
        {ingresses && ingresses.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-muted text-left border-b border-[var(--border-color)]">
                  <th className="py-1.5 pr-3 font-medium">{tl.serviceHealth}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.host}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.path}</th>
                  <th className="py-1.5 pr-3 font-medium">{tl.tls}</th>
                  <th className="py-1.5 font-medium">{tl.backend}</th>
                </tr>
              </thead>
              <tbody>
                {ingresses.map((ing) => (
                  <tr key={ing.name} className="border-b border-[var(--border-color)] last:border-b-0">
                    <td className="py-1.5 pr-3 text-default font-mono">{ing.name}</td>
                    <td className="py-1.5 pr-3 text-default">{ing.hosts.join(", ")}</td>
                    <td className="py-1.5 pr-3 text-muted font-mono">
                      {ing.paths.map(p => p.path).join(", ")}
                    </td>
                    <td className="py-1.5 pr-3">
                      <span className={ing.tlsEnabled ? "text-green-500" : "text-muted"}>
                        {ing.tlsEnabled ? "\u2713" : "\u2717"}
                      </span>
                    </td>
                    <td className="py-1.5 text-muted font-mono">
                      {ing.paths.map(p => `${p.serviceName}:${p.port}`).join(", ")}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : <NoData />}
      </DetailCard>
    </div>
  );
}
