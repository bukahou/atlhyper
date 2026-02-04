"use client";

import { useState, useMemo, useEffect, useCallback, useRef } from "react";
import { Layout } from "@/components/layout/Layout";
import { LoadingSpinner } from "@/components/common";
import { getSLODomainsV2, upsertSLOTarget } from "@/api/slo";
import { getClusterList } from "@/api/cluster";
import {
  Activity,
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  RefreshCw,
  Globe,
  Zap,
  Gauge,
  Server,
  TrendingUp,
  TrendingDown,
  Minus,
  Calendar,
  Box,
  ArrowUpRight,
  ArrowDownRight,
  Settings2,
  X,
  Target,
  CheckCircle2,
  XCircle,
} from "lucide-react";
import type { DomainSLOV2, ServiceSLO, SLOSummary } from "@/types/slo";

// ==================== Types ====================

type TimeRange = "1d" | "7d" | "30d";
type DomainStatus = "healthy" | "warning" | "critical" | "unknown";

// ==================== Components ====================

// 趋势图标
function TrendIcon({ trend }: { trend?: string }) {
  if (trend === "up") return <TrendingUp className="w-4 h-4 text-emerald-500" />;
  if (trend === "down") return <TrendingDown className="w-4 h-4 text-red-500" />;
  return <Minus className="w-4 h-4 text-gray-400" />;
}

// 对比指标组件
function CompareMetric({ label, current, previous, unit, inverse = false }: {
  label: string;
  current: number;
  previous: number;
  unit: string;
  inverse?: boolean;
}) {
  const diff = current - previous;
  const percentDiff = previous !== 0 ? (diff / previous) * 100 : 0;
  const isImproved = inverse ? diff < 0 : diff > 0;
  const isWorsened = inverse ? diff > 0 : diff < 0;

  return (
    <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
      <div className="text-xs text-muted mb-1">{label}</div>
      <div className="flex items-end gap-2">
        <span className="text-lg font-bold text-default">{current.toFixed(2)}{unit}</span>
        <div className={`flex items-center text-xs ${isImproved ? "text-emerald-500" : isWorsened ? "text-red-500" : "text-gray-400"}`}>
          {isImproved ? (
            <ArrowUpRight className="w-3 h-3" />
          ) : isWorsened ? (
            <ArrowDownRight className="w-3 h-3" />
          ) : (
            <Minus className="w-3 h-3" />
          )}
          <span>{Math.abs(percentDiff).toFixed(1)}%</span>
        </div>
      </div>
      <div className="text-xs text-muted mt-0.5">上周期: {previous.toFixed(2)}{unit}</div>
    </div>
  );
}

// SLO 目标配置弹窗
function SLOTargetModal({
  isOpen,
  onClose,
  domain,
  clusterId,
  timeRange,
  onSaved,
}: {
  isOpen: boolean;
  onClose: () => void;
  domain: string;
  clusterId: string;
  timeRange: TimeRange;
  onSaved: () => void;
}) {
  const [selectedRange, setSelectedRange] = useState<TimeRange>(timeRange);
  const [availability, setAvailability] = useState(95);
  const [p95Latency, setP95Latency] = useState(300);
  const [saving, setSaving] = useState(false);

  const errorRateThreshold = (100 - availability).toFixed(2);

  const handleSave = async () => {
    setSaving(true);
    try {
      await upsertSLOTarget({
        clusterId,
        host: domain, // 使用域名作为 host
        timeRange: selectedRange,
        availabilityTarget: availability,
        p95LatencyTarget: p95Latency,
      });
      onSaved();
      onClose();
    } catch (err) {
      console.error("保存 SLO 目标失败:", err);
    } finally {
      setSaving(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* 背景遮罩 */}
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      {/* 弹窗内容 */}
      <div className="relative bg-card border border-[var(--border-color)] rounded-xl shadow-xl w-full max-w-md mx-4">
        {/* 头部 */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <Target className="w-5 h-5 text-primary" />
            <h3 className="font-semibold text-default">配置 SLO 目标</h3>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* 内容 */}
        <div className="p-4 space-y-4">
          {/* 域名显示 */}
          <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
            <div className="text-xs text-muted mb-1">目标域名</div>
            <div className="font-medium text-default">{domain}</div>
          </div>

          {/* 周期选择 */}
          <div>
            <label className="block text-sm font-medium text-default mb-2">选择周期</label>
            <div className="flex gap-2">
              {([
                { value: "1d", label: "天" },
                { value: "7d", label: "周" },
                { value: "30d", label: "月" },
              ] as const).map((range) => (
                <button
                  key={range.value}
                  onClick={() => setSelectedRange(range.value)}
                  className={`flex-1 px-3 py-2 text-sm rounded-lg border transition-colors ${
                    selectedRange === range.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-[var(--border-color)] text-muted hover:text-default hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  {range.label}
                </button>
              ))}
            </div>
          </div>

          {/* 可用性目标 */}
          <div>
            <label className="block text-sm font-medium text-default mb-2">可用性目标 (%)</label>
            <input
              type="number"
              value={availability}
              onChange={(e) => setAvailability(Math.min(100, Math.max(0, Number(e.target.value))))}
              min={0}
              max={100}
              step={0.1}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
            <p className="text-xs text-muted mt-1">默认: 95%，高要求服务可设为 99% 或 99.9%</p>
          </div>

          {/* P95 延迟阈值 */}
          <div>
            <label className="block text-sm font-medium text-default mb-2">P95 延迟阈值 (ms)</label>
            <input
              type="number"
              value={p95Latency}
              onChange={(e) => setP95Latency(Math.max(0, Number(e.target.value)))}
              min={0}
              step={10}
              className="w-full px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default focus:outline-none focus:ring-2 focus:ring-primary/50"
            />
            <p className="text-xs text-muted mt-1">默认: 300ms，高性能服务可设为 100-200ms</p>
          </div>

          {/* 错误率阈值（自动计算） */}
          <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted">错误率阈值</span>
              <span className="text-sm font-medium text-default">{errorRateThreshold}%</span>
            </div>
            <p className="text-xs text-muted mt-1">自动计算: 100% - 可用性目标</p>
          </div>
        </div>

        {/* 底部按钮 */}
        <div className="flex items-center justify-end gap-3 px-4 py-3 border-t border-[var(--border-color)]">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] text-muted hover:text-default hover:bg-[var(--hover-bg)]"
          >
            取消
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-50"
          >
            {saving ? "保存中..." : "保存"}
          </button>
        </div>
      </div>
    </div>
  );
}

// 状态徽章
function StatusBadge({ status }: { status: DomainStatus }) {
  const config = {
    healthy: { bg: "bg-emerald-500/10", text: "text-emerald-500", dot: "bg-emerald-500", label: "健康" },
    warning: { bg: "bg-amber-500/10", text: "text-amber-500", dot: "bg-amber-500", label: "告警" },
    critical: { bg: "bg-red-500/10", text: "text-red-500", dot: "bg-red-500", label: "严重" },
    unknown: { bg: "bg-gray-500/10", text: "text-gray-500", dot: "bg-gray-500", label: "未知" },
  };
  const c = config[status] || config.unknown;
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${c.bg} ${c.text}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${c.dot}`} />
      {c.label}
    </span>
  );
}

// 错误预算条
function ErrorBudgetBar({ percent }: { percent: number }) {
  const isHealthy = percent > 50;
  const isWarning = percent > 20 && percent <= 50;
  return (
    <div className="flex items-center gap-2">
      <div className="flex-1 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all ${
            isHealthy ? "bg-emerald-500" : isWarning ? "bg-amber-500" : "bg-red-500"
          }`}
          style={{ width: `${Math.max(0, Math.min(100, percent))}%` }}
        />
      </div>
      <span className={`text-xs font-medium w-10 text-right ${
        isHealthy ? "text-emerald-500" : isWarning ? "text-amber-500" : "text-red-500"
      }`}>
        {percent.toFixed(0)}%
      </span>
    </div>
  );
}

// 格式化数字
function formatNumber(num: number): string {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`;
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`;
  return num.toLocaleString();
}

// 服务行组件（显示后端服务级别的 SLO - Metrics 的实际数据来源）
function ServiceRow({ service, timeRange }: { service: ServiceSLO; timeRange: TimeRange }) {
  const targets = service.targets?.[timeRange] || { availability: 95, p95_latency: 300 };
  const availability = service.current?.availability ?? 0;
  const p95Latency = service.current?.p95_latency ?? 0;
  const errorRate = service.current?.error_rate ?? 0;
  const rps = service.current?.requests_per_sec ?? 0;

  return (
    <div className="flex items-center gap-4 px-4 py-3 hover:bg-[var(--hover-bg)] transition-colors border-b border-[var(--border-color)] last:border-b-0">
      {/* 服务信息 */}
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <div className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
            service.status === "healthy" ? "bg-emerald-500" :
            service.status === "warning" ? "bg-amber-500" : "bg-red-500"
          }`} />
          <span className="text-sm font-medium text-default">{service.service_name}</span>
          <span className="text-xs text-muted">:{service.service_port}</span>
          <span className="text-xs text-muted">({service.namespace})</span>
        </div>
        {/* 路径列表 */}
        <div className="flex flex-wrap gap-1 ml-3.5">
          {service.paths.map((path, idx) => (
            <code key={idx} className="text-xs font-mono text-muted bg-[var(--hover-bg)] px-1.5 py-0.5 rounded">
              {path}
            </code>
          ))}
        </div>
      </div>

      {/* 指标 */}
      <div className="hidden lg:flex items-center gap-4">
        <div className="w-24 text-right">
          <span className={`text-sm font-medium ${
            availability >= targets.availability ? "text-emerald-500" : "text-red-500"
          }`}>
            {availability.toFixed(2)}%
          </span>
        </div>
        <div className="w-20 text-right">
          <span className={`text-sm font-medium ${
            p95Latency <= targets.p95_latency ? "text-emerald-500" : "text-amber-500"
          }`}>
            {p95Latency}ms
          </span>
        </div>
        <div className="w-20 text-right">
          <span className={`text-sm font-medium ${
            errorRate <= 1 ? "text-emerald-500" : "text-red-500"
          }`}>
            {errorRate.toFixed(2)}%
          </span>
        </div>
        <div className="w-20 text-right">
          <span className="text-sm font-medium text-default">{formatNumber(rps)}/s</span>
        </div>
        <div className="w-16">
          <ErrorBudgetBar percent={service.error_budget_remaining} />
        </div>
      </div>
    </div>
  );
}

// 域名卡片（V2 版本 - 支持域名→路由层级 + 概览 + 周期对比）
function DomainCard({ domain, expanded, onToggle, timeRange, clusterId, onRefresh }: {
  domain: DomainSLOV2;
  expanded: boolean;
  onToggle: () => void;
  timeRange: TimeRange;
  clusterId: string;
  onRefresh: () => void;
}) {
  const [activeTab, setActiveTab] = useState<"slo-status" | "services" | "compare">("slo-status");
  const [showTargetModal, setShowTargetModal] = useState(false);

  // 从 summary 中提取域名级指标
  const availability = domain.summary?.availability ?? 0;
  const p95Latency = domain.summary?.p95_latency ?? 0;
  const p99Latency = domain.summary?.p99_latency ?? 0;
  const errorRate = domain.summary?.error_rate ?? 0;
  const rps = domain.summary?.requests_per_sec ?? 0;
  const totalRequests = domain.summary?.total_requests ?? 0;

  // 计算上周期数据（从所有服务中聚合）
  const prevAvailability = domain.services.reduce((sum, s) => sum + (s.previous?.availability ?? s.current?.availability ?? 0), 0) / Math.max(domain.services.length, 1);
  const prevP95Latency = domain.services.reduce((sum, s) => sum + (s.previous?.p95_latency ?? s.current?.p95_latency ?? 0), 0) / Math.max(domain.services.length, 1);
  const prevErrorRate = domain.services.reduce((sum, s) => sum + (s.previous?.error_rate ?? s.current?.error_rate ?? 0), 0) / Math.max(domain.services.length, 1);

  // 计算趋势
  const trend = availability > prevAvailability ? "up" : availability < prevAvailability ? "down" : "stable";

  // 默认目标值
  const targets = { availability: 95, p95_latency: 300 };

  return (
    <div className="border border-[var(--border-color)] rounded-xl overflow-hidden bg-card">
      {/* 域名摘要行 */}
      <button
        onClick={onToggle}
        className="w-full px-4 py-3 flex items-center gap-4 hover:bg-[var(--hover-bg)] transition-colors"
      >
        {/* 域名信息 */}
        <div className="flex items-center gap-3 flex-1 min-w-0">
          <div className={`p-2 rounded-lg ${
            domain.status === "healthy" ? "bg-emerald-500/10" :
            domain.status === "warning" ? "bg-amber-500/10" : "bg-red-500/10"
          }`}>
            <Globe className={`w-4 h-4 ${
              domain.status === "healthy" ? "text-emerald-500" :
              domain.status === "warning" ? "text-amber-500" : "text-red-500"
            }`} />
          </div>
          <div className="text-left min-w-0">
            <div className="flex items-center gap-2">
              {domain.tls && <span className="text-[10px] text-emerald-600 dark:text-emerald-400 font-medium">HTTPS</span>}
              <span className="font-medium text-default truncate">{domain.domain}</span>
              <StatusBadge status={domain.status as DomainStatus} />
              <span className="text-xs text-muted">({domain.services.length} 个服务)</span>
            </div>
          </div>
        </div>

        {/* 汇总指标 */}
        <div className="hidden lg:flex items-center gap-5">
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">可用性</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                availability >= targets.availability ? "text-emerald-500" : "text-red-500"
              }`}>
                {availability.toFixed(2)}%
              </span>
              <span className="text-xs text-muted">/ {targets.availability}%</span>
            </div>
          </div>
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">P95 延迟</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                p95Latency <= targets.p95_latency ? "text-emerald-500" : "text-amber-500"
              }`}>
                {p95Latency}ms
              </span>
              <span className="text-xs text-muted">/ {targets.p95_latency}ms</span>
            </div>
          </div>
          <div className="w-28">
            <div className="text-[10px] text-muted mb-0.5">错误率</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                errorRate <= 1 ? "text-emerald-500" : "text-red-500"
              }`}>
                {errorRate.toFixed(2)}%
              </span>
            </div>
          </div>
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">错误预算</div>
            <ErrorBudgetBar percent={domain.error_budget_remaining} />
          </div>
          <div className="w-24">
            <div className="text-[10px] text-muted mb-0.5">吞吐量</div>
            <span className="text-sm font-semibold text-default">
              {formatNumber(rps)}/s
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <TrendIcon trend={trend} />
          {expanded ? (
            <ChevronDown className="w-4 h-4 text-muted" />
          ) : (
            <ChevronRight className="w-4 h-4 text-muted" />
          )}
        </div>
      </button>

      {/* 展开详情 */}
      {expanded && (
        <div className="border-t border-[var(--border-color)]">
          {/* Tab 切换 */}
          <div className="flex items-center justify-between px-4 pt-3 pb-2 border-b border-[var(--border-color)]">
            <div className="flex items-center gap-1">
              {[
                { id: "slo-status", label: "SLO 状态", icon: Target },
                { id: "services", label: "服务明细", icon: Box },
                { id: "compare", label: "周期对比", icon: Calendar },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as typeof activeTab)}
                  className={`flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg transition-colors ${
                    activeTab === tab.id
                      ? "bg-primary/10 text-primary"
                      : "text-muted hover:text-default hover:bg-[var(--hover-bg)]"
                  }`}
                >
                  <tab.icon className="w-3.5 h-3.5" />
                  {tab.label}
                </button>
              ))}
            </div>

            {/* 设置按钮 */}
            <button
              onClick={() => setShowTargetModal(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg text-muted hover:text-default hover:bg-[var(--hover-bg)] transition-colors"
            >
              <Settings2 className="w-3.5 h-3.5" />
              配置目标
            </button>
          </div>

          <div className="bg-[var(--background)]">
            {/* SLO 状态 Tab */}
            {activeTab === "slo-status" && (
              <div className="p-4 space-y-5">
                {/* SLO 目标达成情况 */}
                <div>
                  <div className="text-xs font-medium text-muted mb-3">SLO 目标达成情况</div>
                  <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                    {/* 可用性 */}
                    <div className={`p-4 rounded-lg border-2 ${
                      availability >= targets.availability
                        ? "border-emerald-500/30 bg-emerald-500/5"
                        : "border-red-500/30 bg-red-500/5"
                    }`}>
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium text-default">可用性</span>
                        {availability >= targets.availability ? (
                          <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                        ) : (
                          <XCircle className="w-5 h-5 text-red-500" />
                        )}
                      </div>
                      <div className="space-y-1">
                        <div className="flex items-baseline gap-2">
                          <span className="text-2xl font-bold text-default">{availability.toFixed(2)}%</span>
                          <span className="text-xs text-muted">实际</span>
                        </div>
                        <div className="text-xs text-muted">目标: ≥{targets.availability}%</div>
                        <div className={`text-xs font-medium ${
                          availability >= targets.availability ? "text-emerald-500" : "text-red-500"
                        }`}>
                          {availability >= targets.availability
                            ? `✓ 达标 (+${(availability - targets.availability).toFixed(2)}%)`
                            : `✗ 未达标 (${(availability - targets.availability).toFixed(2)}%)`}
                        </div>
                      </div>
                    </div>

                    {/* P95 延迟 */}
                    <div className={`p-4 rounded-lg border-2 ${
                      p95Latency <= targets.p95_latency
                        ? "border-emerald-500/30 bg-emerald-500/5"
                        : "border-amber-500/30 bg-amber-500/5"
                    }`}>
                      <div className="flex items-center justify-between mb-2">
                        <span className="text-sm font-medium text-default">P95 延迟</span>
                        {p95Latency <= targets.p95_latency ? (
                          <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                        ) : (
                          <XCircle className="w-5 h-5 text-amber-500" />
                        )}
                      </div>
                      <div className="space-y-1">
                        <div className="flex items-baseline gap-2">
                          <span className="text-2xl font-bold text-default">{p95Latency}ms</span>
                          <span className="text-xs text-muted">实际</span>
                        </div>
                        <div className="text-xs text-muted">目标: ≤{targets.p95_latency}ms</div>
                        <div className={`text-xs font-medium ${
                          p95Latency <= targets.p95_latency ? "text-emerald-500" : "text-amber-500"
                        }`}>
                          {p95Latency <= targets.p95_latency
                            ? `✓ 达标 (-${targets.p95_latency - p95Latency}ms)`
                            : `✗ 超标 (+${p95Latency - targets.p95_latency}ms)`}
                        </div>
                      </div>
                    </div>

                    {/* 错误率 */}
                    {(() => {
                      const errorRateThreshold = 100 - targets.availability;
                      const isErrorRateOk = errorRate <= errorRateThreshold;
                      return (
                        <div className={`p-4 rounded-lg border-2 ${
                          isErrorRateOk
                            ? "border-emerald-500/30 bg-emerald-500/5"
                            : "border-red-500/30 bg-red-500/5"
                        }`}>
                          <div className="flex items-center justify-between mb-2">
                            <span className="text-sm font-medium text-default">错误率</span>
                            {isErrorRateOk ? (
                              <CheckCircle2 className="w-5 h-5 text-emerald-500" />
                            ) : (
                              <XCircle className="w-5 h-5 text-red-500" />
                            )}
                          </div>
                          <div className="space-y-1">
                            <div className="flex items-baseline gap-2">
                              <span className="text-2xl font-bold text-default">{errorRate.toFixed(2)}%</span>
                              <span className="text-xs text-muted">实际</span>
                            </div>
                            <div className="text-xs text-muted">阈值: ≤{errorRateThreshold.toFixed(2)}%</div>
                            <div className={`text-xs font-medium ${isErrorRateOk ? "text-emerald-500" : "text-red-500"}`}>
                              {isErrorRateOk
                                ? `✓ 达标`
                                : `✗ 超标 (+${(errorRate - errorRateThreshold).toFixed(2)}%)`}
                            </div>
                          </div>
                        </div>
                      );
                    })()}
                  </div>
                </div>

                {/* 错误预算详情 */}
                <div>
                  <div className="text-xs font-medium text-muted mb-3">错误预算详情</div>
                  <div className="p-4 rounded-lg bg-[var(--hover-bg)]">
                    <div className="flex items-center justify-between mb-3">
                      <span className="text-sm text-default">剩余预算</span>
                      <span className={`text-lg font-bold ${
                        domain.error_budget_remaining > 50 ? "text-emerald-500" :
                        domain.error_budget_remaining > 20 ? "text-amber-500" : "text-red-500"
                      }`}>
                        {domain.error_budget_remaining.toFixed(1)}%
                      </span>
                    </div>
                    <div className="h-3 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden mb-3">
                      <div
                        className={`h-full rounded-full transition-all ${
                          domain.error_budget_remaining > 50 ? "bg-emerald-500" :
                          domain.error_budget_remaining > 20 ? "bg-amber-500" : "bg-red-500"
                        }`}
                        style={{ width: `${Math.max(0, Math.min(100, domain.error_budget_remaining))}%` }}
                      />
                    </div>
                    {(() => {
                      const errorRateThreshold = 100 - targets.availability;
                      const allowedErrors = Math.floor(totalRequests * errorRateThreshold / 100);
                      const actualErrors = Math.round(totalRequests * errorRate / 100);
                      const remainingErrors = Math.max(0, allowedErrors - actualErrors);
                      return (
                        <div className="grid grid-cols-3 gap-4 text-center">
                          <div>
                            <div className="text-lg font-bold text-default">{allowedErrors}</div>
                            <div className="text-xs text-muted">允许错误数</div>
                          </div>
                          <div>
                            <div className={`text-lg font-bold ${actualErrors > allowedErrors ? "text-red-500" : "text-amber-500"}`}>
                              {actualErrors}
                            </div>
                            <div className="text-xs text-muted">已发生错误</div>
                          </div>
                          <div>
                            <div className={`text-lg font-bold ${remainingErrors > 0 ? "text-emerald-500" : "text-red-500"}`}>
                              {remainingErrors}
                            </div>
                            <div className="text-xs text-muted">剩余配额</div>
                          </div>
                        </div>
                      );
                    })()}
                  </div>
                </div>

                {/* 流量统计 */}
                <div>
                  <div className="text-xs font-medium text-muted mb-3">流量统计</div>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                    <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                      <div className="text-xs text-muted mb-1">总请求数</div>
                      <div className="text-lg font-bold text-default">{formatNumber(totalRequests)}</div>
                    </div>
                    <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                      <div className="text-xs text-muted mb-1">成功请求</div>
                      <div className="text-lg font-bold text-emerald-500">
                        {formatNumber(Math.round(totalRequests * (1 - errorRate / 100)))}
                      </div>
                      <div className="text-xs text-muted">{(100 - errorRate).toFixed(2)}%</div>
                    </div>
                    <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                      <div className="text-xs text-muted mb-1">错误请求</div>
                      <div className={`text-lg font-bold ${errorRate > 0 ? "text-red-500" : "text-default"}`}>
                        {formatNumber(Math.round(totalRequests * errorRate / 100))}
                      </div>
                      <div className="text-xs text-muted">{errorRate.toFixed(2)}%</div>
                    </div>
                    <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                      <div className="text-xs text-muted mb-1">平均吞吐量</div>
                      <div className="text-lg font-bold text-default">{rps.toFixed(2)}/s</div>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* 服务明细 Tab */}
            {activeTab === "services" && (
              <div>
                {/* 服务列表头 */}
                <div className="flex items-center gap-4 px-4 py-2 text-xs text-muted border-b border-[var(--border-color)] bg-[var(--hover-bg)]">
                  <div className="flex-1">后端服务（Metrics 数据来源）</div>
                  <div className="hidden lg:flex items-center gap-4">
                    <div className="w-24 text-right">可用性</div>
                    <div className="w-20 text-right">P95 延迟</div>
                    <div className="w-20 text-right">错误率</div>
                    <div className="w-20 text-right">吞吐量</div>
                    <div className="w-16">错误预算</div>
                  </div>
                </div>

                {/* 服务列表（支持滚动） */}
                {domain.services.length > 0 ? (
                  <div className="max-h-80 overflow-y-auto">
                    {domain.services.map((service, idx) => (
                      <ServiceRow
                        key={`${service.service_key}-${idx}`}
                        service={service}
                        timeRange={timeRange}
                      />
                    ))}
                  </div>
                ) : (
                  <div className="px-4 py-6 text-center text-sm text-muted">
                    暂无服务数据
                  </div>
                )}

                {/* 服务数量提示 */}
                {domain.services.length > 3 && (
                  <div className="px-4 py-2 text-xs text-muted border-t border-[var(--border-color)] bg-[var(--hover-bg)]">
                    共 {domain.services.length} 个后端服务
                  </div>
                )}
              </div>
            )}

            {/* 周期对比 Tab */}
            {activeTab === "compare" && (
              <div className="p-4 space-y-4">
                <div className="flex items-center gap-2 text-xs text-muted">
                  <Calendar className="w-4 h-4" />
                  <span>本周期 vs 上周期对比</span>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <CompareMetric
                    label="可用性"
                    current={availability}
                    previous={prevAvailability}
                    unit="%"
                    inverse={false}
                  />
                  <CompareMetric
                    label="P95 延迟"
                    current={p95Latency}
                    previous={prevP95Latency}
                    unit="ms"
                    inverse={true}
                  />
                  <CompareMetric
                    label="错误率"
                    current={errorRate}
                    previous={prevErrorRate}
                    unit="%"
                    inverse={true}
                  />
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* SLO 目标配置弹窗 */}
      <SLOTargetModal
        isOpen={showTargetModal}
        onClose={() => setShowTargetModal(false)}
        domain={domain.domain}
        clusterId={clusterId}
        timeRange={timeRange}
        onSaved={onRefresh}
      />
    </div>
  );
}

// 汇总卡片
function SummaryCard({
  icon: Icon,
  label,
  value,
  subValue,
  color,
}: {
  icon: typeof Activity;
  label: string;
  value: string;
  subValue?: string;
  color: string;
}) {
  return (
    <div className="p-4 rounded-xl bg-card border border-[var(--border-color)]">
      <div className="flex items-center gap-3">
        <div className={`p-2 rounded-lg ${color}`}>
          <Icon className="w-5 h-5" />
        </div>
        <div>
          <div className="text-xs text-muted">{label}</div>
          <div className="text-xl font-bold text-default">{value}</div>
          {subValue && <div className="text-xs text-muted">{subValue}</div>}
        </div>
      </div>
    </div>
  );
}

// ==================== Main Page ====================

const REFRESH_INTERVAL = 30000;

export default function SLOPage() {
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [domains, setDomains] = useState<DomainSLOV2[]>([]);
  const [summary, setSummary] = useState<SLOSummary | null>(null);
  const [clusterId, setClusterId] = useState("");

  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>("1d");

  const isMountedRef = useRef(true);
  const isFirstLoadRef = useRef(true);

  // 获取数据
  const fetchData = useCallback(async (showRefreshing = false) => {
    if (showRefreshing) setRefreshing(true);

    try {
      let currentClusterId = clusterId;
      if (!currentClusterId) {
        const clusterRes = await getClusterList();
        const clusters = clusterRes.data?.clusters || [];
        if (clusters.length === 0) {
          if (isMountedRef.current && isFirstLoadRef.current) {
            setError("暂无可用集群");
          }
          return;
        }
        currentClusterId = clusters[0].cluster_id;
        setClusterId(currentClusterId);
      }

      const res = await getSLODomainsV2({ clusterId: currentClusterId, timeRange });
      if (isMountedRef.current) {
        setDomains(res.data?.domains || []);
        setSummary(res.data?.summary || null);
        setError("");
      }
    } catch (err) {
      if (isMountedRef.current) {
        console.warn("[SLO] Fetch error:", err);
        if (isFirstLoadRef.current) {
          setError(err instanceof Error ? err.message : "加载失败");
        }
      }
    } finally {
      if (isMountedRef.current) {
        setLoading(false);
        setRefreshing(false);
        isFirstLoadRef.current = false;
      }
    }
  }, [clusterId, timeRange]);

  // 初始加载和自动刷新
  useEffect(() => {
    isMountedRef.current = true;
    fetchData();

    const intervalId = setInterval(() => {
      fetchData(true);
    }, REFRESH_INTERVAL);

    return () => {
      isMountedRef.current = false;
      clearInterval(intervalId);
    };
  }, [fetchData]);

  // 手动刷新
  const handleRefresh = () => {
    fetchData(true);
  };

  // 计算汇总数据（从 API 返回或本地计算）
  const summaryData = useMemo(() => {
    if (summary) {
      return {
        totalDomains: summary.total_domains,
        healthyCount: summary.healthy_count,
        warningCount: summary.warning_count,
        criticalCount: summary.critical_count,
        totalRPS: summary.total_rps,
        avgAvailability: summary.avg_availability,
        avgErrorBudget: summary.avg_error_budget,
      };
    }

    // 本地计算（V2 结构：domain.summary）
    const totalDomains = domains.length;
    const healthyCount = domains.filter(d => d.status === "healthy").length;
    const warningCount = domains.filter(d => d.status === "warning").length;
    const criticalCount = domains.filter(d => d.status === "critical").length;
    const totalRPS = domains.reduce((sum, d) => sum + (d.summary?.requests_per_sec || 0), 0);
    const avgAvailability = totalDomains > 0
      ? domains.reduce((sum, d) => sum + (d.summary?.availability || 0), 0) / totalDomains
      : 0;
    const avgErrorBudget = totalDomains > 0
      ? domains.reduce((sum, d) => sum + (d.error_budget_remaining || 0), 0) / totalDomains
      : 0;

    return {
      totalDomains,
      healthyCount,
      warningCount,
      criticalCount,
      totalRPS,
      avgAvailability,
      avgErrorBudget,
    };
  }, [domains, summary]);

  if (loading) {
    return (
      <Layout>
        <LoadingSpinner />
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="-m-6 min-h-[calc(100vh-3.5rem)] bg-[var(--background)]">
        {/* 头部 */}
        <div className="px-6 py-4 border-b border-[var(--border-color)] bg-card">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-xl bg-gradient-to-br from-violet-100 to-indigo-100 dark:from-violet-900/30 dark:to-indigo-900/30">
                <Activity className="w-6 h-6 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <h1 className="text-lg font-semibold text-default">SLO 监控</h1>
                <p className="text-xs text-muted">按域名维度查看可用性、延迟和错误率指标</p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              {/* 时间范围选择 */}
              <div className="flex items-center gap-1 p-1 rounded-lg bg-[var(--hover-bg)]">
                {([
                  { value: "1d", label: "天" },
                  { value: "7d", label: "周" },
                  { value: "30d", label: "月" },
                ] as const).map((range) => (
                  <button
                    key={range.value}
                    onClick={() => setTimeRange(range.value)}
                    className={`px-3 py-1 text-xs rounded-md transition-colors ${
                      timeRange === range.value
                        ? "bg-card text-default shadow-sm"
                        : "text-muted hover:text-default"
                    }`}
                  >
                    {range.label}
                  </button>
                ))}
              </div>
              <button
                onClick={handleRefresh}
                disabled={refreshing}
                className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
              >
                <RefreshCw className={`w-4 h-4 ${refreshing ? "animate-spin" : ""}`} />
              </button>
            </div>
          </div>
        </div>

        <div className="p-6 space-y-6">
          {/* Error State */}
          {error && domains.length === 0 && (
            <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
              <AlertTriangle className="w-12 h-12 mx-auto mb-3 text-red-500" />
              <p className="text-red-500">{error}</p>
            </div>
          )}

          {/* Empty State */}
          {!error && domains.length === 0 && (
            <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
              <Server className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
              <p className="text-default font-medium mb-2">暂无 SLO 数据</p>
              <p className="text-sm text-muted">请确保 Agent 已启用 SLO 采集并正确配置 Ingress Controller</p>
            </div>
          )}

          {/* 有数据时显示 */}
          {domains.length > 0 && (
            <>
              {/* 汇总卡片 */}
              <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
                <SummaryCard
                  icon={Globe}
                  label="监控域名"
                  value={summaryData.totalDomains.toString()}
                  subValue={`${summaryData.healthyCount} 健康`}
                  color="bg-blue-500/10 text-blue-500"
                />
                <SummaryCard
                  icon={Activity}
                  label="平均可用性"
                  value={`${summaryData.avgAvailability.toFixed(2)}%`}
                  color="bg-emerald-500/10 text-emerald-500"
                />
                <SummaryCard
                  icon={Gauge}
                  label="错误预算剩余"
                  value={`${summaryData.avgErrorBudget.toFixed(0)}%`}
                  subValue="平均剩余"
                  color={summaryData.avgErrorBudget > 50 ? "bg-emerald-500/10 text-emerald-500" : summaryData.avgErrorBudget > 20 ? "bg-amber-500/10 text-amber-500" : "bg-red-500/10 text-red-500"}
                />
                <SummaryCard
                  icon={Zap}
                  label="总吞吐量"
                  value={formatNumber(summaryData.totalRPS)}
                  subValue="req/s"
                  color="bg-violet-500/10 text-violet-500"
                />
                <SummaryCard
                  icon={AlertTriangle}
                  label="告警中"
                  value={summaryData.warningCount.toString()}
                  subValue="需要关注"
                  color="bg-amber-500/10 text-amber-500"
                />
                <SummaryCard
                  icon={AlertTriangle}
                  label="严重问题"
                  value={summaryData.criticalCount.toString()}
                  subValue="需立即处理"
                  color="bg-red-500/10 text-red-500"
                />
              </div>

              {/* 域名 SLO 列表 */}
              <div>
                <div className="flex items-center justify-between mb-4">
                  <h2 className="text-sm font-semibold text-default">
                    域名 SLO 状态
                    <span className="ml-2 text-xs font-normal text-muted">({summaryData.totalDomains} 个域名)</span>
                  </h2>
                </div>
                <div className="space-y-3">
                  {domains.map((domain) => (
                    <DomainCard
                      key={domain.domain}
                      domain={domain}
                      expanded={expandedId === domain.domain}
                      onToggle={() => setExpandedId(expandedId === domain.domain ? null : domain.domain)}
                      timeRange={timeRange}
                      clusterId={clusterId}
                      onRefresh={handleRefresh}
                    />
                  ))}
                </div>
              </div>

              {/* 说明 */}
              <div className="p-4 rounded-xl bg-blue-50 dark:bg-blue-950/30 border border-blue-200 dark:border-blue-800">
                <div className="flex items-start gap-3">
                  <div className="p-1.5 rounded-lg bg-blue-100 dark:bg-blue-900/50">
                    <Activity className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                  </div>
                  <div className="text-sm">
                    <p className="font-medium text-blue-800 dark:text-blue-200 mb-1">数据来源说明</p>
                    <p className="text-blue-700 dark:text-blue-300 text-xs leading-relaxed">
                      所有指标均基于 Ingress Controller 流量数据计算，按域名（Host）维度聚合。
                      系统采集 Traefik/Nginx/Kong 的 Prometheus 指标，计算可用性、延迟百分位数和错误率。
                    </p>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>
    </Layout>
  );
}
