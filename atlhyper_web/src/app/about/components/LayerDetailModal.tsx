"use client";

import { Modal } from "@/components/common";
import { CheckCircle2, Clock } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { AboutTranslations } from "@/types/i18n";

interface LayerDetailModalProps {
  isOpen: boolean;
  onClose: () => void;
  a: AboutTranslations;
  layer: {
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
  };
  colorStyle: { bg: string; text: string; border: string; badge: string };
}

// 每个段落
function Section({ title, children, accent }: { title: string; children: React.ReactNode; accent?: string }) {
  return (
    <div>
      <h4 className={`text-sm font-semibold mb-1.5 ${accent || "text-default"}`}>{title}</h4>
      <div className="text-sm text-secondary leading-relaxed">{children}</div>
    </div>
  );
}

export function LayerDetailModal({ isOpen, onClose, a, layer, colorStyle: c }: LayerDetailModalProps) {
  const Icon = layer.icon;
  const statusDone = layer.status === "done";

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`${a[layer.titleKey]}`}
      size="md"
    >
      <div className="p-6 space-y-5">
        {/* 顶部：图标 + 概要 */}
        <div className="flex items-start gap-4">
          <div className={`w-12 h-12 rounded-xl ${c.bg} flex items-center justify-center flex-shrink-0`}>
            <Icon className={`w-6 h-6 ${c.text}`} />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-1">
              <span className={`text-xs font-bold ${c.text}`}>{layer.level}</span>
              {statusDone ? (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-500/15 text-emerald-500">
                  <CheckCircle2 className="w-3 h-3" />{a.statusDone}
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-500/15 text-blue-500">
                  <Clock className="w-3 h-3" />{a.statusPlanned}
                </span>
              )}
            </div>
            <p className="text-sm text-secondary">{a[layer.descKey]}</p>
            <div className="mt-2 flex flex-wrap gap-1.5">
              {String(a[layer.metricsKey]).split(" · ").map((m) => (
                <span key={m} className={`px-2 py-0.5 rounded text-xs ${c.badge}`}>{m}</span>
              ))}
            </div>
          </div>
        </div>

        <hr className="border-[var(--border-color)]/50" />

        {/* 概念 */}
        <Section title={a.detailSectionWhat}>
          {a[layer.detailWhatKey]}
        </Section>

        {/* 作用 */}
        <Section title={a.detailSectionRole}>
          {a[layer.detailRoleKey]}
        </Section>

        {/* 行业现状 */}
        <Section title={a.detailSectionIndustry}>
          {a[layer.detailIndustryKey]}
        </Section>

        {/* 主流方案 */}
        <Section title={a.detailSectionTools}>
          <div className="flex flex-wrap gap-1.5 mt-1">
            {String(a[layer.detailToolsKey]).split(" · ").map((tool) => (
              <span
                key={tool}
                className="px-2.5 py-1 rounded-lg bg-card border border-[var(--border-color)] text-xs font-medium text-secondary"
              >
                {tool}
              </span>
            ))}
          </div>
        </Section>

        {/* AtlHyper 实现 */}
        <Section title={a.detailSectionAtlhyper} accent={c.text}>
          <div className={`p-3 rounded-lg ${c.bg} border ${c.border}`}>
            <p className="text-sm text-secondary leading-relaxed">{a[layer.detailAtlhyperKey]}</p>
          </div>
        </Section>

        {/* 数据来源 */}
        <div className="pt-2 border-t border-[var(--border-color)]/50">
          <span className={`inline-flex items-center gap-1 px-2.5 py-1 rounded ${c.badge} text-xs font-mono`}>
            {a[layer.sourceKey]}
          </span>
        </div>
      </div>
    </Modal>
  );
}
