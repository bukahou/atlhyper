"use client";

import { StatusBadge } from "@/components/common";
import type { Translations } from "@/types/i18n";
import type {
  IngressDetail,
  IngressRuleDTO,
  IngressTLSDTO,
} from "@/types/cluster";
import {
  Globe,
  Lock,
  Server,
  ArrowRight,
} from "lucide-react";

// 概览 Tab
export function OverviewTab({ detail, t }: { detail: IngressDetail; t: Translations }) {
  const infoItems = [
    { label: t.common.name, value: detail.name },
    { label: t.common.namespace, value: detail.namespace },
    { label: t.ingress.ingressClass, value: detail.class || detail.spec?.ingressClassName || "-" },
    { label: "Controller", value: detail.controller || "-" },
    { label: t.ingress.age, value: detail.age || "-" },
    { label: t.common.createdAt, value: detail.createdAt ? new Date(detail.createdAt).toLocaleString() : "-" },
  ];

  return (
    <div className="space-y-6">
      {/* 基本信息 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.ingress.basicInfo}</h3>
        <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
          {infoItems.map((item, i) => (
            <div key={i} className="bg-[var(--background)] rounded-lg p-3">
              <div className="text-xs text-muted mb-1">{item.label}</div>
              <div className="text-sm text-default font-medium">{item.value}</div>
            </div>
          ))}
        </div>
      </div>

      {/* TLS 状态 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.ingress.tlsStatus}</h3>
        <div className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center gap-3">
            {detail.tlsEnabled ? (
              <>
                <Lock className="w-5 h-5 text-green-500" />
                <span className="text-green-600 font-medium">{t.ingress.tlsEnabled}</span>
                <StatusBadge status="Enabled" type="success" />
              </>
            ) : (
              <>
                <Lock className="w-5 h-5 text-muted" />
                <span className="text-muted">{t.ingress.tlsDisabled}</span>
                <StatusBadge status="Disabled" type="default" />
              </>
            )}
          </div>
        </div>
      </div>

      {/* Hosts */}
      {detail.hosts && detail.hosts.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">Hosts</h3>
          <div className="flex flex-wrap gap-2">
            {detail.hosts.map((host, i) => (
              <span key={i} className="px-3 py-1.5 bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 text-sm font-mono rounded">
                {host}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* LoadBalancer */}
      {detail.loadBalancer && detail.loadBalancer.length > 0 && (
        <div>
          <h3 className="text-sm font-semibold text-default mb-3">LoadBalancer IPs</h3>
          <div className="flex flex-wrap gap-2">
            {detail.loadBalancer.map((ip, i) => (
              <span key={i} className="px-3 py-1.5 bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 text-sm font-mono rounded">
                {ip}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* 规则统计 */}
      <div>
        <h3 className="text-sm font-semibold text-default mb-3">{t.ingress.ruleStatistics}</h3>
        <div className="grid grid-cols-3 gap-4">
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-primary">{detail.spec?.rules?.length || 0}</div>
            <div className="text-xs text-muted mt-1">{t.ingress.routingRules}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-blue-500">
              {detail.spec?.rules?.reduce((sum, r) => sum + (r.paths?.length || 0), 0) || 0}
            </div>
            <div className="text-xs text-muted mt-1">{t.ingress.pathCount}</div>
          </div>
          <div className="bg-[var(--background)] rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-green-500">{detail.spec?.tls?.length || 0}</div>
            <div className="text-xs text-muted mt-1">{t.ingress.tlsCertificates}</div>
          </div>
        </div>
      </div>
    </div>
  );
}

// Backend 显示组件
function BackendDisplay({ backend }: { backend: IngressDetail["spec"]["defaultBackend"] }) {
  if (!backend) return <span className="text-muted">-</span>;

  if (backend.type === "Service" && backend.service) {
    const port = backend.service.portName || backend.service.portNumber;
    return (
      <span className="font-mono text-sm text-default">
        {backend.service.name}{port ? `:${port}` : ""}
      </span>
    );
  }

  if (backend.type === "Resource" && backend.resource) {
    return (
      <span className="font-mono text-sm text-default">
        {backend.resource.kind}/{backend.resource.name}
      </span>
    );
  }

  return <span className="text-muted">-</span>;
}

// 路由规则 Tab
export function RulesTab({ rules, defaultBackend, t }: { rules: IngressRuleDTO[]; defaultBackend?: IngressDetail["spec"]["defaultBackend"]; t: Translations }) {
  if (rules.length === 0 && !defaultBackend) {
    return <div className="text-center py-8 text-muted">{t.ingress.noRules}</div>;
  }

  return (
    <div className="space-y-4">
      {/* Default Backend */}
      {defaultBackend && (
        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
          <h4 className="font-medium text-yellow-700 dark:text-yellow-400 mb-2">{t.ingress.defaultBackend}</h4>
          <BackendDisplay backend={defaultBackend} />
        </div>
      )}

      {/* Rules */}
      {rules.map((rule, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center gap-2 mb-4">
            <Globe className="w-4 h-4 text-primary" />
            <h4 className="font-medium text-default">{rule.host || "*"}</h4>
          </div>

          <div className="space-y-3">
            {rule.paths.map((path, j) => (
              <div key={j} className="flex items-center gap-3 p-3 bg-card rounded-lg border border-[var(--border-color)]">
                <div className="flex-1 flex items-center gap-3">
                  <span className="px-2 py-1 text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded">
                    {path.pathType || "Prefix"}
                  </span>
                  <span className="font-mono text-sm text-default">{path.path || "/"}</span>
                </div>
                <ArrowRight className="w-4 h-4 text-muted" />
                <div className="flex items-center gap-2">
                  <Server className="w-4 h-4 text-muted" />
                  <BackendDisplay backend={path.backend} />
                </div>
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

// TLS Tab
export function TLSTab({ tls, t }: { tls: IngressTLSDTO[]; t: Translations }) {
  if (tls.length === 0) {
    return <div className="text-center py-8 text-muted">{t.ingress.noTlsConfig}</div>;
  }

  return (
    <div className="space-y-3">
      {tls.map((item, i) => (
        <div key={i} className="bg-[var(--background)] rounded-lg p-4">
          <div className="flex items-center gap-2 mb-3">
            <Lock className="w-4 h-4 text-green-500" />
            <h4 className="font-medium text-default">{item.secretName}</h4>
          </div>

          {item.hosts && item.hosts.length > 0 && (
            <div>
              <div className="text-xs text-muted mb-2">Hosts:</div>
              <div className="flex flex-wrap gap-2">
                {item.hosts.map((host, j) => (
                  <span key={j} className="px-2 py-1 text-sm font-mono bg-card border border-[var(--border-color)] rounded">
                    {host}
                  </span>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// 注解 Tab
export function AnnotationsTab({ annotations, t }: { annotations: Record<string, string>; t: Translations }) {
  const entries = Object.entries(annotations);

  if (entries.length === 0) {
    return <div className="text-center py-8 text-muted">{t.ingress.noAnnotations}</div>;
  }

  return (
    <div className="space-y-2">
      {entries.map(([key, value]) => (
        <div key={key} className="bg-[var(--background)] rounded-lg p-3">
          <div className="text-sm font-mono text-primary break-all mb-1">{key}</div>
          <div className="text-sm font-mono text-default break-all">{value || '""'}</div>
        </div>
      ))}
    </div>
  );
}
