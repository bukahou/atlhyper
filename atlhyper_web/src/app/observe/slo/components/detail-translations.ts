// detail-translations.ts
// DomainDetail 及其子组件的翻译契约。
// 单独一个文件是因为它被 DomainDetail / BurnRateTable / GoodBadCount / LatencyTab /
// SLOTargetModal 五处共用，写在组件里会导致循环引用。

import type { SLOTranslations } from "@/types/i18n";
import type { BurnRateTableTranslations } from "./BurnRateTable";
import type { GoodBadCountTranslations } from "./GoodBadCount";

export interface DomainDetailTranslations
  extends BurnRateTableTranslations,
    GoodBadCountTranslations {
  tabBudget: string;
  tabLatency: string;
  configTarget: string;
  // 图表
  p95Latency: string;
  errorRate: string;
  target: string;
  sloTrend: string;
  errorBudgetBurn: string;
  current: string;
  estimatedExhaust: string;
  // 延迟 tab
  latencyDistribution: string;
  methodBreakdown: string;
  statusCodeBreakdown: string;
  clearSelection: string;
  requests: string;
  // 目标配置弹窗
  configSloTarget: string;
  targetDomain: string;
  sloWindow: string;
  sloWindowHint: string;
  days: string;
  targetAvailability: string;
  targetAvailabilityHint: string;
  targetP95: string;
  targetP95Hint: string;
  errorRateThreshold: string;
  errorRateAutoCalc: string;
  cancel: string;
  save: string;
  saving: string;
}

/**
 * 从 SLO 页的翻译里挑出详情用的那一部分。
 *
 * 清单表那一行已经承载了域名 / 目标 / SLI / 预算 / 燃烧率概要，这里只需要详情独有的部分。
 */
export function buildDomainDetailTranslations(sloT: SLOTranslations): DomainDetailTranslations {
  return {
    tabBudget: sloT.tabOverview,
    tabLatency: sloT.tabLatency,
    configTarget: sloT.configTarget,
    // 燃烧率表
    burnRate: sloT.burnRate,
    burnRateHint: sloT.burnRateHint,
    window: sloT.window,
    rate: sloT.rate,
    threshold: sloT.threshold,
    status: sloT.sloStatus,
    good: sloT.healthy,
    warn: sloT.warning,
    crit: sloT.critical,
    // Good/Bad 计数
    eventCount: sloT.eventCount,
    bad: sloT.badEvents,
    allowed: sloT.allowedEvents,
    overspent: sloT.overspent,
    // 图表
    p95Latency: sloT.p95Latency,
    errorRate: sloT.errorRate,
    target: sloT.target,
    sloTrend: sloT.sloTrend,
    errorBudgetBurn: sloT.errorBudgetBurn,
    current: sloT.current,
    estimatedExhaust: sloT.estimatedExhaust,
    noData: sloT.noData,
    // 延迟 tab
    latencyDistribution: sloT.latencyDistribution,
    methodBreakdown: sloT.methodBreakdown,
    statusCodeBreakdown: sloT.statusCodeBreakdown,
    clearSelection: sloT.clearSelection,
    requests: sloT.requests,
    // 目标配置
    configSloTarget: sloT.configSloTarget,
    targetDomain: sloT.targetDomain,
    sloWindow: sloT.sloWindow,
    sloWindowHint: sloT.sloWindowHint,
    days: sloT.days,
    targetAvailability: sloT.targetAvailability,
    targetAvailabilityHint: sloT.targetAvailabilityHint,
    targetP95: sloT.targetP95,
    targetP95Hint: sloT.targetP95Hint,
    errorRateThreshold: sloT.errorRateThreshold,
    errorRateAutoCalc: sloT.errorRateAutoCalc,
    cancel: sloT.cancel,
    save: sloT.save,
    saving: sloT.saving,
  };
}
