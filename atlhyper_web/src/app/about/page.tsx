"use client";

import { useState } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import {
  Globe,
  Network,
  Search,
  FileText,
  Cpu,
  Box,
  Activity,
  Bot,
  ClipboardList,
  AlertTriangle,
  BookOpen,
  CheckCircle2,
  Clock,
  ChevronDown,
  Zap,
  Server,
  Monitor,
  Github,
  Scale,
  ExternalLink,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { AboutTranslations } from "@/types/i18n";
import { LayerDetailModal } from "./components";

// ── 五层架构定义 ──

interface LayerDef {
  level: string;
  titleKey: keyof AboutTranslations;
  descKey: keyof AboutTranslations;
  sourceKey: keyof AboutTranslations;
  metricsKey: keyof AboutTranslations;
  icon: LucideIcon;
  status: "done" | "planned";
  color: string;
  detailWhatKey: keyof AboutTranslations;
  detailRoleKey: keyof AboutTranslations;
  detailIndustryKey: keyof AboutTranslations;
  detailToolsKey: keyof AboutTranslations;
  detailAtlhyperKey: keyof AboutTranslations;
}

const layers: LayerDef[] = [
  { level: "L1", titleKey: "layer1Title", descKey: "layer1Desc", sourceKey: "layer1Source", metricsKey: "layer1Metrics", icon: Globe, status: "done", color: "blue", detailWhatKey: "layer1DetailWhat", detailRoleKey: "layer1DetailRole", detailIndustryKey: "layer1DetailIndustry", detailToolsKey: "layer1DetailTools", detailAtlhyperKey: "layer1DetailAtlhyper" },
  { level: "L2", titleKey: "layer2Title", descKey: "layer2Desc", sourceKey: "layer2Source", metricsKey: "layer2Metrics", icon: Network, status: "done", color: "violet", detailWhatKey: "layer2DetailWhat", detailRoleKey: "layer2DetailRole", detailIndustryKey: "layer2DetailIndustry", detailToolsKey: "layer2DetailTools", detailAtlhyperKey: "layer2DetailAtlhyper" },
  { level: "L3", titleKey: "layer3Title", descKey: "layer3Desc", sourceKey: "layer3Source", metricsKey: "layer3Metrics", icon: Search, status: "planned", color: "amber", detailWhatKey: "layer3DetailWhat", detailRoleKey: "layer3DetailRole", detailIndustryKey: "layer3DetailIndustry", detailToolsKey: "layer3DetailTools", detailAtlhyperKey: "layer3DetailAtlhyper" },
  { level: "L4", titleKey: "layer4Title", descKey: "layer4Desc", sourceKey: "layer4Source", metricsKey: "layer4Metrics", icon: FileText, status: "planned", color: "emerald", detailWhatKey: "layer4DetailWhat", detailRoleKey: "layer4DetailRole", detailIndustryKey: "layer4DetailIndustry", detailToolsKey: "layer4DetailTools", detailAtlhyperKey: "layer4DetailAtlhyper" },
  { level: "L5", titleKey: "layer5Title", descKey: "layer5Desc", sourceKey: "layer5Source", metricsKey: "layer5Metrics", icon: Cpu, status: "done", color: "rose", detailWhatKey: "layer5DetailWhat", detailRoleKey: "layer5DetailRole", detailIndustryKey: "layer5DetailIndustry", detailToolsKey: "layer5DetailTools", detailAtlhyperKey: "layer5DetailAtlhyper" },
];

const drilldowns: { key: keyof AboutTranslations }[] = [
  { key: "drilldown12" },
  { key: "drilldown23" },
  { key: "drilldown34" },
  { key: "drilldown45" },
];

// ── 功能模块定义 ──

interface FeatureDef {
  icon: LucideIcon;
  titleKey: keyof AboutTranslations;
  descKey: keyof AboutTranslations;
  status: "done" | "planned";
}

const featureModules: FeatureDef[] = [
  { icon: Box, titleKey: "featureClusterTitle", descKey: "featureClusterDesc", status: "done" },
  { icon: Activity, titleKey: "featureSloTitle", descKey: "featureSloDesc", status: "done" },
  { icon: Network, titleKey: "featureTopologyTitle", descKey: "featureTopologyDesc", status: "done" },
  { icon: Bot, titleKey: "featureAiTitle", descKey: "featureAiDesc", status: "done" },
  { icon: ClipboardList, titleKey: "featureCommandTitle", descKey: "featureCommandDesc", status: "done" },
  { icon: AlertTriangle, titleKey: "featureAlertTitle", descKey: "featureAlertDesc", status: "done" },
  { icon: Cpu, titleKey: "featureMetricsTitle", descKey: "featureMetricsDesc", status: "done" },
  { icon: Search, titleKey: "featureApmTitle", descKey: "featureApmDesc", status: "planned" },
  { icon: FileText, titleKey: "featureLogsTitle", descKey: "featureLogsDesc", status: "planned" },
];

// ── 技术栈定义 ──

interface TechDef {
  icon: LucideIcon;
  titleKey: keyof AboutTranslations;
  stackKey: keyof AboutTranslations;
  descKey: keyof AboutTranslations;
}

const techStack: TechDef[] = [
  { icon: Server, titleKey: "techMasterTitle", stackKey: "techMasterStack", descKey: "techMasterDesc" },
  { icon: Zap, titleKey: "techAgentTitle", stackKey: "techAgentStack", descKey: "techAgentDesc" },
  { icon: Cpu, titleKey: "techMetricsTitle", stackKey: "techMetricsStack", descKey: "techMetricsDesc" },
  { icon: Monitor, titleKey: "techWebTitle", stackKey: "techWebStack", descKey: "techWebDesc" },
];

// ── 组件 ──

function StatusBadge({ status, label }: { status: "done" | "planned"; label: string }) {
  if (status === "done") {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-500 flex-shrink-0">
        <CheckCircle2 className="w-3 h-3" />
        {label}
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-500/15 text-blue-500 flex-shrink-0">
      <Clock className="w-3 h-3" />
      {label}
    </span>
  );
}

// 层级颜色映射
const colorMap: Record<string, { bg: string; text: string; border: string; badge: string }> = {
  blue:    { bg: "bg-blue-500/10",    text: "text-blue-500",    border: "border-blue-500/20",    badge: "bg-blue-500/20 text-blue-400" },
  violet:  { bg: "bg-violet-500/10",  text: "text-violet-500",  border: "border-violet-500/20",  badge: "bg-violet-500/20 text-violet-400" },
  amber:   { bg: "bg-amber-500/10",   text: "text-amber-500",   border: "border-amber-500/20",   badge: "bg-amber-500/20 text-amber-400" },
  emerald: { bg: "bg-emerald-500/10", text: "text-emerald-500", border: "border-emerald-500/20", badge: "bg-emerald-500/20 text-emerald-400" },
  rose:    { bg: "bg-rose-500/10",    text: "text-rose-500",    border: "border-rose-500/20",    badge: "bg-rose-500/20 text-rose-400" },
};

export default function AboutPage() {
  const { t } = useI18n();
  const a = t.aboutPage;
  const [selectedLayer, setSelectedLayer] = useState<number | null>(null);

  const statusLabel = (s: "done" | "planned") =>
    s === "done" ? a.statusDone : a.statusPlanned;

  return (
    <Layout>
      <div className="max-w-5xl mx-auto space-y-12 pb-16">

        {/* ═══ Hero ═══ */}
        <section className="text-center pt-8 pb-2">
          <h1 className="text-4xl font-bold text-default tracking-tight">AtlHyper</h1>
          <p className="mt-3 text-lg text-primary font-medium">{a.subtitle}</p>
          <p className="mt-3 text-sm text-muted max-w-2xl mx-auto leading-relaxed">
            {a.description}
          </p>
          {/* 关键数字 */}
          <div className="mt-8 grid grid-cols-2 sm:grid-cols-4 gap-4 max-w-2xl mx-auto">
            {[
              { value: "4", label: a.heroStatComponents },
              { value: "5", label: a.heroStatLayers },
              { value: "2", label: a.heroStatLanguages },
              { value: "MIT", label: a.heroStatLicense },
            ].map((stat) => (
              <div key={stat.label} className="bg-card rounded-xl border border-[var(--border-color)] p-4">
                <div className="text-2xl font-bold text-primary">{stat.value}</div>
                <div className="text-xs text-muted mt-1">{stat.label}</div>
              </div>
            ))}
          </div>
        </section>

        {/* ═══ 五层可观测性架构 ═══ */}
        <section>
          <div className="flex items-center gap-2 mb-2">
            <BookOpen className="w-5 h-5 text-primary" />
            <h2 className="text-xl font-semibold text-default">{a.sectionArchitecture}</h2>
          </div>
          <p className="text-sm text-muted mb-1">{a.sectionArchitectureDesc}</p>
          <p className="text-xs text-muted/60 mb-6 flex items-center gap-1">
            <ExternalLink className="w-3 h-3" />
            {a.detailClickHint}
          </p>

          {/* 五层 + drill-down */}
          <div className="space-y-0">
            {layers.map((layer, idx) => {
              const Icon = layer.icon;
              const c = colorMap[layer.color];
              const drill = idx < drilldowns.length ? drilldowns[idx] : null;

              return (
                <div key={layer.level}>
                  {/* 层级卡片 — 可点击 */}
                  <button
                    type="button"
                    onClick={() => setSelectedLayer(idx)}
                    className={`w-full text-left bg-card rounded-xl border ${c.border} p-5 relative cursor-pointer transition-all duration-150 hover:shadow-md hover:border-opacity-60 hover:scale-[1.005] active:scale-[0.998]`}
                  >
                    <div className="flex items-start gap-4">
                      {/* 层级标号 + 图标 */}
                      <div className="flex flex-col items-center gap-1 flex-shrink-0">
                        <span className={`text-xs font-bold ${c.text} tracking-wide`}>{layer.level}</span>
                        <div className={`w-10 h-10 rounded-lg ${c.bg} flex items-center justify-center`}>
                          <Icon className={`w-5 h-5 ${c.text}`} />
                        </div>
                      </div>
                      {/* 内容 */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <h3 className="text-base font-semibold text-default">{a[layer.titleKey]}</h3>
                          <StatusBadge status={layer.status} label={statusLabel(layer.status)} />
                        </div>
                        <p className="text-sm text-secondary mb-2">{a[layer.descKey]}</p>
                        <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-4 text-xs text-muted">
                          <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded ${c.badge} text-xs font-mono`}>
                            {a[layer.sourceKey]}
                          </span>
                        </div>
                        <p className="text-xs text-muted mt-2 leading-relaxed">{a[layer.metricsKey]}</p>
                      </div>
                    </div>
                  </button>

                  {/* Drill-down 箭头 */}
                  {drill && (
                    <div className="flex items-center gap-2 py-2 pl-6">
                      <ChevronDown className="w-4 h-4 text-primary/60" />
                      <span className="text-xs font-medium text-primary/80">{a.drilldownLabel}</span>
                      <span className="text-xs text-muted">{a[drill.key]}</span>
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          {/* AI 全链路面板 */}
          <div className="mt-6 bg-gradient-to-br from-primary/5 via-card to-primary/5 rounded-xl border border-primary/20 p-6">
            <div className="flex items-center gap-3 mb-3">
              <div className="w-10 h-10 rounded-lg bg-primary/15 flex items-center justify-center">
                <Bot className="w-5 h-5 text-primary" />
              </div>
              <div>
                <h3 className="text-base font-semibold text-default">{a.aiTitle}</h3>
              </div>
            </div>
            <p className="text-sm text-secondary leading-relaxed mb-4">{a.aiDesc}</p>

            {/* 场景示例 */}
            <div className="bg-card/80 rounded-lg border border-[var(--border-color)] p-4">
              <h4 className="text-sm font-semibold text-default mb-3 flex items-center gap-2">
                <Zap className="w-4 h-4 text-amber-500" />
                {a.aiScenarioTitle}
              </h4>
              <div className="space-y-2">
                {[
                  { step: a.aiScenarioStep1, color: colorMap.blue },
                  { step: a.aiScenarioStep2, color: colorMap.violet },
                  { step: a.aiScenarioStep3, color: colorMap.amber },
                  { step: a.aiScenarioStep4, color: colorMap.emerald },
                  { step: a.aiScenarioStep5, color: colorMap.rose },
                ].map((item, idx) => (
                  <div key={idx} className="flex items-start gap-2">
                    <span className={`w-5 h-5 rounded-full ${item.color.bg} ${item.color.text} flex items-center justify-center text-xs font-bold flex-shrink-0 mt-0.5`}>
                      {idx + 1}
                    </span>
                    <span className="text-xs text-secondary leading-relaxed">{item.step}</span>
                  </div>
                ))}
              </div>
              <div className="mt-3 pt-3 border-t border-[var(--border-color)]/50">
                <p className="text-xs font-medium text-primary">{a.aiFullStack}</p>
              </div>
            </div>
          </div>
        </section>

        {/* ═══ 功能模块 ═══ */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Box className="w-5 h-5 text-primary" />
            <h2 className="text-xl font-semibold text-default">{a.sectionFeatures}</h2>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {featureModules.map((mod) => {
              const Icon = mod.icon;
              return (
                <div
                  key={mod.titleKey}
                  className="bg-card rounded-xl border border-[var(--border-color)] p-4 flex items-start gap-3"
                >
                  <div className="w-9 h-9 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0 mt-0.5">
                    <Icon className="w-4 h-4 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <h3 className="text-sm font-semibold text-default truncate">{a[mod.titleKey]}</h3>
                      <StatusBadge status={mod.status} label={statusLabel(mod.status)} />
                    </div>
                    <p className="text-xs text-muted leading-relaxed">{a[mod.descKey]}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </section>

        {/* ═══ 技术架构 ═══ */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Cpu className="w-5 h-5 text-primary" />
            <h2 className="text-xl font-semibold text-default">{a.sectionTechStack}</h2>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {techStack.map((tech) => {
              const Icon = tech.icon;
              return (
                <div
                  key={tech.titleKey}
                  className="bg-card rounded-xl border border-[var(--border-color)] p-4"
                >
                  <div className="flex items-center gap-2 mb-2">
                    <Icon className="w-4 h-4 text-primary" />
                    <h3 className="text-sm font-bold text-default">{a[tech.titleKey]}</h3>
                  </div>
                  <p className="text-xs font-mono text-primary/80 mb-1">{a[tech.stackKey]}</p>
                  <p className="text-xs text-muted leading-relaxed">{a[tech.descKey]}</p>
                </div>
              );
            })}
          </div>
        </section>

        {/* ═══ 开源信息 ═══ */}
        <section>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-6 flex flex-col sm:flex-row items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center">
                <Scale className="w-5 h-5 text-primary" />
              </div>
              <div>
                <h3 className="text-base font-semibold text-default">{a.sectionOpenSource}</h3>
                <p className="text-sm text-muted">{a.openSourceDesc}</p>
                <p className="text-xs text-muted mt-1">{a.openSourceRequirements}</p>
              </div>
            </div>
            <a
              href="https://github.com/bukahou/atlhyper"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-primary/10 text-primary text-sm font-medium hover:bg-primary/20 transition-colors flex-shrink-0"
            >
              <Github className="w-4 h-4" />
              GitHub
            </a>
          </div>
        </section>

      </div>

      {/* ═══ 层级详情弹窗 ═══ */}
      {selectedLayer !== null && (
        <LayerDetailModal
          isOpen
          onClose={() => setSelectedLayer(null)}
          a={a}
          layer={layers[selectedLayer]}
          colorStyle={colorMap[layers[selectedLayer].color]}
        />
      )}
    </Layout>
  );
}
