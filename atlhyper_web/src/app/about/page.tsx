"use client";

import { useState } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import {
  Box,
  Cpu,
  BookOpen,
  ChevronDown,
  Zap,
  Bot,
  Github,
  Scale,
  ExternalLink,
  BrainCircuit,
  ArrowRight,
} from "lucide-react";
import {
  LayerDetailModal,
  StatusBadge,
  layers,
  drilldowns,
  featureModules,
  techStack,
  aiopsCaps,
  colorMap,
} from "./components";

export default function AboutPage() {
  const { t } = useI18n();
  const a = t.aboutPage;
  const [selectedLayer, setSelectedLayer] = useState<number | null>(null);

  const statusLabel = (s: "done" | "planned") =>
    s === "done" ? a.statusDone : a.statusPlanned;

  return (
    <Layout>
      <div className="max-w-5xl mx-auto space-y-12 pb-16">

        {/* Hero */}
        <section className="text-center pt-8 pb-2">
          <h1 className="text-4xl font-bold text-default tracking-tight">AtlHyper</h1>
          <p className="mt-3 text-lg text-primary font-medium">{a.subtitle}</p>
          <p className="mt-3 text-sm text-muted max-w-2xl mx-auto leading-relaxed">
            {a.description}
          </p>
          <div className="mt-8 grid grid-cols-2 sm:grid-cols-4 gap-4 max-w-2xl mx-auto">
            {[
              { value: "3", label: a.heroStatComponents },
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

        {/* Architecture */}
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

          <div className="space-y-0">
            {layers.map((layer, idx) => {
              const Icon = layer.icon;
              const c = colorMap[layer.color];
              const drill = idx < drilldowns.length ? drilldowns[idx] : null;
              return (
                <div key={layer.level}>
                  <button
                    type="button"
                    onClick={() => setSelectedLayer(idx)}
                    className={`w-full text-left bg-card rounded-xl border ${c.border} p-5 relative cursor-pointer transition-all duration-150 hover:shadow-md hover:border-opacity-60 hover:scale-[1.005] active:scale-[0.998]`}
                  >
                    <div className="flex items-start gap-4">
                      <div className="flex flex-col items-center gap-1 flex-shrink-0">
                        <span className={`text-xs font-bold ${c.text} tracking-wide`}>{layer.level}</span>
                        <div className={`w-10 h-10 rounded-lg ${c.bg} flex items-center justify-center`}>
                          <Icon className={`w-5 h-5 ${c.text}`} />
                        </div>
                      </div>
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

          {/* AI Panel */}
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

        {/* AIOps Engine */}
        <section>
          <div className="flex items-center gap-2 mb-2">
            <BrainCircuit className="w-5 h-5 text-amber-500" />
            <h2 className="text-xl font-semibold text-default">{a.aiopsEngineTitle}</h2>
          </div>
          <p className="text-sm text-muted mb-6">{a.aiopsEngineDesc}</p>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {aiopsCaps.map((cap) => {
              const Icon = cap.icon;
              const c = colorMap[cap.color];
              return (
                <div key={cap.titleKey} className={`bg-card rounded-xl border ${c.border} p-5`}>
                  <div className="flex items-start gap-3">
                    <div className={`w-10 h-10 rounded-lg ${c.bg} flex items-center justify-center flex-shrink-0`}>
                      <Icon className={`w-5 h-5 ${c.text}`} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1">
                        <h3 className="text-sm font-semibold text-default">{a[cap.titleKey]}</h3>
                        <span className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-amber-500/15 text-amber-400">
                          {a.aiopsStatusPartial}
                        </span>
                      </div>
                      <p className="text-xs text-muted leading-relaxed">{a[cap.descKey]}</p>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Pipeline flow */}
          <div className="mt-5 bg-card/50 rounded-xl border border-[var(--border-color)] p-4">
            <div className="flex flex-wrap items-center justify-center gap-2 text-xs text-secondary">
              {[a.aiopsFlow1, a.aiopsFlow2, a.aiopsFlow3, a.aiopsFlow4].map((step, idx) => (
                <span key={idx} className="flex items-center gap-2">
                  <span className="px-2.5 py-1 rounded-lg bg-amber-500/10 text-amber-400 font-medium">{step}</span>
                  {idx < 3 && <ArrowRight className="w-3.5 h-3.5 text-muted" />}
                </span>
              ))}
            </div>
            <p className="text-center text-xs text-muted mt-3">{a.aiopsVision}</p>
          </div>
        </section>

        {/* Features */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Box className="w-5 h-5 text-primary" />
            <h2 className="text-xl font-semibold text-default">{a.sectionFeatures}</h2>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {featureModules.map((mod) => {
              const Icon = mod.icon;
              return (
                <div key={mod.titleKey} className="bg-card rounded-xl border border-[var(--border-color)] p-4 flex items-start gap-3">
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

        {/* Tech Stack */}
        <section>
          <div className="flex items-center gap-2 mb-4">
            <Cpu className="w-5 h-5 text-primary" />
            <h2 className="text-xl font-semibold text-default">{a.sectionTechStack}</h2>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {techStack.map((tech) => {
              const Icon = tech.icon;
              return (
                <div key={tech.titleKey} className="bg-card rounded-xl border border-[var(--border-color)] p-4">
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

        {/* Open Source */}
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

      {/* Layer detail modal */}
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
