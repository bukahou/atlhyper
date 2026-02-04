"use client";

import { useState, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import {
  Activity,
  AlertTriangle,
  TrendingUp,
  TrendingDown,
  Minus,
  ChevronDown,
  ChevronRight,
  RefreshCw,
  Settings2,
  Globe,
  Zap,
  Gauge,
  X,
  Download,
  ArrowUpRight,
  ArrowDownRight,
  Calendar,
  FileText,
  Target,
} from "lucide-react";

// ==================== Types ====================

// 历史数据点
interface HistoryPoint {
  timestamp: string;
  availability: number;
  p95Latency: number;
  p99Latency: number;
  errorRate: number;
  rps: number;
  errorBudgetRemaining: number;
}

// SLO 目标
interface SLOTargets {
  availability: number;
  p95Latency: number;
  errorRate: number;
}

// 时间周期类型
type TimeRange = "1d" | "7d" | "30d";

// 域名级别的 SLO 数据
interface DomainSLO {
  id: string;
  host: string;
  ingressName: string;
  ingressClass: string;
  namespace: string;
  tls: boolean;
  // 按时间周期区分的目标
  targets: {
    "1d": SLOTargets;
    "7d": SLOTargets;
    "30d": SLOTargets;
  };
  current: {
    availability: number;
    p95Latency: number;
    p99Latency: number;
    errorRate: number;
    requestsPerSec: number;
    totalRequests: number;
  };
  // 上周期对比
  previous: {
    availability: number;
    p95Latency: number;
    errorRate: number;
  };
  errorBudgetRemaining: number;
  status: "healthy" | "warning" | "critical";
  trend: "up" | "down" | "stable";
  history: HistoryPoint[];
}

// ==================== Mock Data ====================

// 生成历史数据
function generateHistory(days: number, baseAvail: number, baseLatency: number): HistoryPoint[] {
  const points: HistoryPoint[] = [];
  const now = new Date();
  for (let i = days * 24; i >= 0; i -= 4) {
    const timestamp = new Date(now.getTime() - i * 60 * 60 * 1000).toISOString();
    const noise = Math.random() * 0.3 - 0.15;
    const latencyNoise = Math.random() * 50 - 25;
    points.push({
      timestamp,
      availability: Math.min(100, Math.max(95, baseAvail + noise)),
      p95Latency: Math.max(10, baseLatency + latencyNoise),
      p99Latency: Math.max(20, baseLatency * 1.8 + latencyNoise * 2),
      errorRate: Math.max(0, (100 - baseAvail - noise) * 0.8),
      rps: Math.floor(Math.random() * 500 + 1500),
      errorBudgetRemaining: Math.max(0, 70 + Math.random() * 30 - i * 0.02),
    });
  }
  return points;
}

// 默认 SLO 目标
const defaultTargets: SLOTargets = { availability: 95, p95Latency: 300, errorRate: 5 };

const mockDomainSLOs: DomainSLO[] = [
  {
    id: "1",
    host: "api.example.com",
    ingressName: "api-gateway",
    ingressClass: "nginx",
    namespace: "production",
    tls: true,
    targets: {
      "1d": { availability: 95, p95Latency: 300, errorRate: 5 },
      "7d": { availability: 96, p95Latency: 280, errorRate: 4 },
      "30d": { availability: 97, p95Latency: 250, errorRate: 3 },
    },
    current: { availability: 99.74, p95Latency: 226, p99Latency: 520, errorRate: 0.26, requestsPerSec: 1790, totalRequests: 77600000 },
    previous: { availability: 99.82, p95Latency: 198, errorRate: 0.18 },
    errorBudgetRemaining: 65,
    status: "warning",
    trend: "stable",
    history: generateHistory(7, 99.74, 226),
  },
  {
    id: "2",
    host: "pay.example.com",
    ingressName: "payment-gateway",
    ingressClass: "nginx",
    namespace: "finance",
    tls: true,
    targets: {
      "1d": { availability: 99, p95Latency: 100, errorRate: 1 },
      "7d": { availability: 99.5, p95Latency: 100, errorRate: 0.5 },
      "30d": { availability: 99.9, p95Latency: 100, errorRate: 0.1 },
    },
    current: { availability: 99.92, p95Latency: 125, p99Latency: 245, errorRate: 0.08, requestsPerSec: 95, totalRequests: 4120000 },
    previous: { availability: 99.96, p95Latency: 95, errorRate: 0.04 },
    errorBudgetRemaining: 12,
    status: "critical",
    trend: "down",
    history: generateHistory(7, 99.92, 125),
  },
  {
    id: "3",
    host: "www.example.com",
    ingressName: "frontend",
    ingressClass: "nginx",
    namespace: "production",
    tls: true,
    targets: {
      "1d": { availability: 95, p95Latency: 500, errorRate: 5 },
      "7d": { availability: 96, p95Latency: 450, errorRate: 4 },
      "30d": { availability: 97, p95Latency: 400, errorRate: 3 },
    },
    current: { availability: 99.95, p95Latency: 292, p99Latency: 546, errorRate: 0.05, requestsPerSec: 4250, totalRequests: 184200000 },
    previous: { availability: 99.91, p95Latency: 310, errorRate: 0.09 },
    errorBudgetRemaining: 85,
    status: "healthy",
    trend: "up",
    history: generateHistory(7, 99.95, 292),
  },
  {
    id: "4",
    host: "admin.example.com",
    ingressName: "admin-portal",
    ingressClass: "nginx",
    namespace: "internal",
    tls: true,
    targets: {
      "1d": { availability: 90, p95Latency: 800, errorRate: 10 },
      "7d": { availability: 92, p95Latency: 700, errorRate: 8 },
      "30d": { availability: 95, p95Latency: 600, errorRate: 5 },
    },
    current: { availability: 99.85, p95Latency: 450, p99Latency: 920, errorRate: 0.15, requestsPerSec: 25, totalRequests: 1080000 },
    previous: { availability: 99.78, p95Latency: 480, errorRate: 0.22 },
    errorBudgetRemaining: 92,
    status: "healthy",
    trend: "stable",
    history: generateHistory(7, 99.85, 450),
  },
  {
    id: "5",
    host: "static.example.com",
    ingressName: "cdn-origin",
    ingressClass: "nginx",
    namespace: "infra",
    tls: true,
    targets: {
      "1d": { availability: 95, p95Latency: 50, errorRate: 5 },
      "7d": { availability: 96, p95Latency: 50, errorRate: 4 },
      "30d": { availability: 97, p95Latency: 50, errorRate: 3 },
    },
    current: { availability: 99.99, p95Latency: 22, p99Latency: 45, errorRate: 0.01, requestsPerSec: 12500, totalRequests: 542000000 },
    previous: { availability: 99.98, p95Latency: 25, errorRate: 0.02 },
    errorBudgetRemaining: 98,
    status: "healthy",
    trend: "up",
    history: generateHistory(7, 99.99, 22),
  },
];

// ==================== Components ====================

// 趋势图标
function TrendIcon({ trend }: { trend: "up" | "down" | "stable" }) {
  if (trend === "up") return <TrendingUp className="w-4 h-4 text-emerald-500" />;
  if (trend === "down") return <TrendingDown className="w-4 h-4 text-red-500" />;
  return <Minus className="w-4 h-4 text-gray-400" />;
}

// 状态徽章
function StatusBadge({ status }: { status: "healthy" | "warning" | "critical" }) {
  const config = {
    healthy: { bg: "bg-emerald-500/10", text: "text-emerald-500", dot: "bg-emerald-500" },
    warning: { bg: "bg-amber-500/10", text: "text-amber-500", dot: "bg-amber-500" },
    critical: { bg: "bg-red-500/10", text: "text-red-500", dot: "bg-red-500" },
  };
  const c = config[status];
  return (
    <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${c.bg} ${c.text}`}>
      <span className={`w-1.5 h-1.5 rounded-full ${c.dot}`} />
      {status === "healthy" ? "健康" : status === "warning" ? "告警" : "严重"}
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

// 对比指标组件
function CompareMetric({ label, current, previous, unit, inverse = false }: {
  label: string;
  current: number;
  previous: number;
  unit: string;
  inverse?: boolean; // 是否反向（如延迟，越低越好）
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

// 格式化请求数（千分位分隔）
function formatNumber(num: number): string {
  return num.toLocaleString();
}

// 域名卡片
function DomainCard({ domain, expanded, onToggle, timeRange, onEditTargets }: {
  domain: DomainSLO;
  expanded: boolean;
  onToggle: () => void;
  timeRange: TimeRange;
  onEditTargets: () => void;
}) {
  const [activeTab, setActiveTab] = useState<"overview" | "compare">("overview");

  // 获取当前时间周期的目标
  const targets = domain.targets[timeRange];

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
              <span className="font-medium text-default truncate">{domain.host}</span>
              <StatusBadge status={domain.status} />
            </div>
            <div className="text-xs text-muted flex items-center gap-2 mt-0.5">
              <span>{domain.namespace}/{domain.ingressName}</span>
              <span className="text-gray-400">·</span>
              <span>{domain.ingressClass}</span>
            </div>
          </div>
        </div>

        {/* 汇总指标 */}
        <div className="hidden lg:flex items-center gap-5">
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">可用性</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                domain.current.availability >= targets.availability ? "text-emerald-500" : "text-red-500"
              }`}>
                {domain.current.availability.toFixed(2)}%
              </span>
              <span className="text-xs text-muted">/ {targets.availability}%</span>
            </div>
          </div>
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">P95 延迟</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                domain.current.p95Latency <= targets.p95Latency ? "text-emerald-500" : "text-amber-500"
              }`}>
                {domain.current.p95Latency}ms
              </span>
              <span className="text-xs text-muted">/ {targets.p95Latency}ms</span>
            </div>
          </div>
          <div className="w-28">
            <div className="text-[10px] text-muted mb-0.5">错误率</div>
            <div className="flex items-center gap-1">
              <span className={`text-sm font-semibold ${
                domain.current.errorRate <= targets.errorRate ? "text-emerald-500" : "text-red-500"
              }`}>
                {domain.current.errorRate.toFixed(2)}%
              </span>
              <span className="text-xs text-muted">/ {targets.errorRate}%</span>
            </div>
          </div>
          <div className="w-32">
            <div className="text-[10px] text-muted mb-0.5">错误预算</div>
            <ErrorBudgetBar percent={domain.errorBudgetRemaining} />
          </div>
          <div className="w-24">
            <div className="text-[10px] text-muted mb-0.5">吞吐量</div>
            <span className="text-sm font-semibold text-default">
              {formatNumber(domain.current.requestsPerSec)}/s
            </span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <TrendIcon trend={domain.trend} />
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
          <div className="flex items-center gap-1 px-4 pt-3 pb-2 border-b border-[var(--border-color)]">
            {[
              { id: "overview", label: "概览", icon: Activity },
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
            <button
              onClick={onEditTargets}
              className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg text-muted hover:text-default hover:bg-[var(--hover-bg)] transition-colors"
            >
              <Settings2 className="w-3.5 h-3.5" />
              编辑目标
            </button>
          </div>

          <div className="p-4 bg-[var(--background)]">
            {/* 概览 Tab */}
            {activeTab === "overview" && (
              <div className="space-y-4">
                {/* 核心指标 */}
                <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">可用性</div>
                    <div className="text-lg font-bold text-default">{domain.current.availability.toFixed(3)}%</div>
                    <div className="text-xs text-muted mt-1">目标: {targets.availability}%</div>
                  </div>
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">P95 / P99 延迟</div>
                    <div className="text-lg font-bold text-default">{domain.current.p95Latency}ms / {domain.current.p99Latency}ms</div>
                    <div className="text-xs text-muted mt-1">目标 P95: {targets.p95Latency}ms</div>
                  </div>
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">错误率</div>
                    <div className="text-lg font-bold text-default">{domain.current.errorRate.toFixed(3)}%</div>
                    <div className="text-xs text-muted mt-1">目标: {targets.errorRate}%</div>
                  </div>
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">总请求数</div>
                    <div className="text-lg font-bold text-default">{formatNumber(domain.current.totalRequests)}</div>
                    <div className="text-xs text-muted mt-1">{formatNumber(domain.current.requestsPerSec)} req/s</div>
                  </div>
                  <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
                    <div className="text-xs text-muted mb-1">错误预算剩余</div>
                    <div className={`text-lg font-bold ${
                      domain.errorBudgetRemaining > 50 ? "text-emerald-500" :
                      domain.errorBudgetRemaining > 20 ? "text-amber-500" : "text-red-500"
                    }`}>
                      {domain.errorBudgetRemaining.toFixed(1)}%
                    </div>
                    <div className="mt-1">
                      <ErrorBudgetBar percent={domain.errorBudgetRemaining} />
                    </div>
                  </div>
                </div>
              </div>
            )}

            {/* 周期对比 Tab */}
            {activeTab === "compare" && (
              <div className="space-y-4">
                <div className="flex items-center gap-2 text-xs text-muted">
                  <Calendar className="w-4 h-4" />
                  <span>本周期 vs 上周期对比</span>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <CompareMetric
                    label="可用性"
                    current={domain.current.availability}
                    previous={domain.previous.availability}
                    unit="%"
                    inverse={false}
                  />
                  <CompareMetric
                    label="P95 延迟"
                    current={domain.current.p95Latency}
                    previous={domain.previous.p95Latency}
                    unit="ms"
                    inverse={true}
                  />
                  <CompareMetric
                    label="错误率"
                    current={domain.current.errorRate}
                    previous={domain.previous.errorRate}
                    unit="%"
                    inverse={true}
                  />
                </div>
              </div>
            )}

          </div>
        </div>
      )}
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

// SLO 配置弹窗
function SLOConfigModal({ domain, onClose, onSave, currentTimeRange }: {
  domain: DomainSLO;
  onClose: () => void;
  onSave: (timeRange: TimeRange, targets: SLOTargets) => void;
  currentTimeRange: TimeRange;
}) {
  const [selectedRange, setSelectedRange] = useState<TimeRange>(currentTimeRange);
  const [targets, setTargets] = useState(domain.targets[selectedRange]);

  // 切换时间周期时更新目标值
  const handleRangeChange = (range: TimeRange) => {
    setSelectedRange(range);
    setTargets(domain.targets[range]);
  };

  const timeRangeLabels: Record<TimeRange, string> = {
    "1d": "天",
    "7d": "周",
    "30d": "月",
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-md bg-card rounded-2xl shadow-xl border border-[var(--border-color)] overflow-hidden">
        <div className="flex items-center justify-between p-4 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <Target className="w-5 h-5 text-primary" />
            <h3 className="font-semibold text-default">配置 SLO 目标</h3>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--hover-bg)]">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <div className="p-4 space-y-4">
          <div className="text-sm text-muted mb-2">
            <Globe className="w-4 h-4 inline mr-1" />
            {domain.host}
          </div>

          {/* 时间周期选择 */}
          <div>
            <label className="text-sm font-medium text-default mb-2 block">选择周期</label>
            <div className="flex gap-2">
              {(["1d", "7d", "30d"] as TimeRange[]).map((range) => (
                <button
                  key={range}
                  onClick={() => handleRangeChange(range)}
                  className={`flex-1 px-3 py-2 text-sm rounded-lg border transition-colors ${
                    selectedRange === range
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-[var(--border-color)] text-muted hover:text-default"
                  }`}
                >
                  {timeRangeLabels[range]}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="text-sm font-medium text-default">可用性目标 (%)</label>
            <input
              type="number"
              step="0.01"
              min="90"
              max="100"
              value={targets.availability}
              onChange={(e) => setTargets({ ...targets, availability: parseFloat(e.target.value) })}
              className="w-full mt-1 px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default"
            />
            <div className="text-xs text-muted mt-1">默认: 95%，高要求服务可设为 99% 或 99.9%</div>
          </div>

          <div>
            <label className="text-sm font-medium text-default">P95 延迟阈值 (ms)</label>
            <input
              type="number"
              step="10"
              min="10"
              max="5000"
              value={targets.p95Latency}
              onChange={(e) => setTargets({ ...targets, p95Latency: parseInt(e.target.value) })}
              className="w-full mt-1 px-3 py-2 rounded-lg border border-[var(--border-color)] bg-[var(--background)] text-default"
            />
            <div className="text-xs text-muted mt-1">默认: 300ms，高性能服务可设为 100-200ms</div>
          </div>

          {/* 错误率自动计算显示 */}
          <div className="p-3 rounded-lg bg-[var(--hover-bg)]">
            <div className="flex items-center justify-between">
              <span className="text-sm text-muted">错误率阈值</span>
              <span className="text-sm font-medium text-default">{(100 - targets.availability).toFixed(2)}%</span>
            </div>
            <div className="text-xs text-muted mt-1">自动计算: 100% - 可用性目标</div>
          </div>
        </div>

        <div className="flex items-center gap-2 p-4 border-t border-[var(--border-color)] bg-[var(--hover-bg)]">
          <button
            onClick={onClose}
            className="flex-1 px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] text-muted hover:text-default transition-colors"
          >
            取消
          </button>
          <button
            onClick={() => onSave(selectedRange, {
              ...targets,
              errorRate: 100 - targets.availability, // 自动计算错误率
            })}
            className="flex-1 px-4 py-2 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 transition-colors"
          >
            保存 ({timeRangeLabels[selectedRange]})
          </button>
        </div>
      </div>
    </div>
  );
}

// 报告导出弹窗
function ExportReportModal({ onClose }: { onClose: () => void }) {
  const [period, setPeriod] = useState<"week" | "month">("week");
  const [format, setFormat] = useState<"pdf" | "csv" | "json">("pdf");
  const [exporting, setExporting] = useState(false);

  const handleExport = () => {
    setExporting(true);
    setTimeout(() => {
      setExporting(false);
      onClose();
      // 模拟下载
      alert(`已导出 ${period === "week" ? "周" : "月"}报 (${format.toUpperCase()} 格式)`);
    }, 1500);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-md bg-card rounded-2xl shadow-xl border border-[var(--border-color)] overflow-hidden">
        <div className="flex items-center justify-between p-4 border-b border-[var(--border-color)]">
          <div className="flex items-center gap-2">
            <FileText className="w-5 h-5 text-primary" />
            <h3 className="font-semibold text-default">导出 SLO 报告</h3>
          </div>
          <button onClick={onClose} className="p-1 rounded-lg hover:bg-[var(--hover-bg)]">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        <div className="p-4 space-y-4">
          <div>
            <label className="text-sm font-medium text-default mb-2 block">报告周期</label>
            <div className="flex gap-2">
              {[
                { value: "week", label: "本周报告" },
                { value: "month", label: "本月报告" },
              ].map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setPeriod(opt.value as typeof period)}
                  className={`flex-1 px-4 py-2 text-sm rounded-lg border transition-colors ${
                    period === opt.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-[var(--border-color)] text-muted hover:text-default"
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <div>
            <label className="text-sm font-medium text-default mb-2 block">导出格式</label>
            <div className="flex gap-2">
              {[
                { value: "pdf", label: "PDF" },
                { value: "csv", label: "CSV" },
                { value: "json", label: "JSON" },
              ].map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => setFormat(opt.value as typeof format)}
                  className={`flex-1 px-4 py-2 text-sm rounded-lg border transition-colors ${
                    format === opt.value
                      ? "border-primary bg-primary/10 text-primary"
                      : "border-[var(--border-color)] text-muted hover:text-default"
                  }`}
                >
                  {opt.label}
                </button>
              ))}
            </div>
          </div>

          <div className="p-3 rounded-lg bg-[var(--hover-bg)] text-xs text-muted">
            <p className="font-medium text-default mb-1">报告内容包括：</p>
            <ul className="space-y-0.5 list-disc list-inside">
              <li>各域名 SLO 达成情况汇总</li>
              <li>错误预算使用详情</li>
              <li>趋势分析与改进建议</li>
            </ul>
          </div>
        </div>

        <div className="flex items-center gap-2 p-4 border-t border-[var(--border-color)] bg-[var(--hover-bg)]">
          <button
            onClick={onClose}
            className="flex-1 px-4 py-2 text-sm rounded-lg border border-[var(--border-color)] text-muted hover:text-default transition-colors"
          >
            取消
          </button>
          <button
            onClick={handleExport}
            disabled={exporting}
            className="flex-1 px-4 py-2 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
          >
            {exporting ? (
              <>
                <RefreshCw className="w-4 h-4 animate-spin" />
                导出中...
              </>
            ) : (
              <>
                <Download className="w-4 h-4" />
                导出报告
              </>
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

// ==================== Main Page ====================

export default function StylePreviewPage() {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [timeRange, setTimeRange] = useState<TimeRange>("1d");
  const [showConfigModal, setShowConfigModal] = useState<DomainSLO | null>(null);
  const [showExportModal, setShowExportModal] = useState(false);
  const [domains, setDomains] = useState(mockDomainSLOs);

  // 计算汇总数据
  const summary = useMemo(() => {
    const totalDomains = domains.length;
    const healthyCount = domains.filter(d => d.status === "healthy").length;
    const warningCount = domains.filter(d => d.status === "warning").length;
    const criticalCount = domains.filter(d => d.status === "critical").length;
    const totalRPS = domains.reduce((sum, d) => sum + d.current.requestsPerSec, 0);
    const avgAvailability = domains.reduce((sum, d) => sum + d.current.availability, 0) / totalDomains;
    const avgErrorBudget = domains.reduce((sum, d) => sum + d.errorBudgetRemaining, 0) / totalDomains;

    return {
      totalDomains,
      healthyCount,
      warningCount,
      criticalCount,
      totalRPS,
      avgAvailability,
      avgErrorBudget,
    };
  }, [domains]);

  const handleSaveConfig = (domainId: string, range: TimeRange, newTargets: SLOTargets) => {
    setDomains(prev => prev.map(d =>
      d.id === domainId ? {
        ...d,
        targets: {
          ...d.targets,
          [range]: newTargets,
        }
      } : d
    ));
    setShowConfigModal(null);
  };

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
                <h1 className="text-lg font-semibold text-default">样式设计</h1>
                <p className="text-xs text-muted">UI 组件样式设计预览（SLO 监控示例）</p>
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
                onClick={() => setShowExportModal(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 text-xs rounded-lg bg-[var(--hover-bg)] text-muted hover:text-default transition-colors"
              >
                <Download className="w-3.5 h-3.5" />
                导出报告
              </button>
              <button className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors">
                <RefreshCw className="w-4 h-4" />
              </button>
              <button className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors">
                <Settings2 className="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        <div className="p-6 space-y-6">
          {/* 汇总卡片 */}
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            <SummaryCard
              icon={Globe}
              label="监控域名"
              value={summary.totalDomains.toString()}
              subValue={`${summary.healthyCount} 健康`}
              color="bg-blue-500/10 text-blue-500"
            />
            <SummaryCard
              icon={Activity}
              label="平均可用性"
              value={`${summary.avgAvailability.toFixed(2)}%`}
              color="bg-emerald-500/10 text-emerald-500"
            />
            <SummaryCard
              icon={Gauge}
              label="错误预算剩余"
              value={`${summary.avgErrorBudget.toFixed(0)}%`}
              subValue="平均剩余"
              color={summary.avgErrorBudget > 50 ? "bg-emerald-500/10 text-emerald-500" : summary.avgErrorBudget > 20 ? "bg-amber-500/10 text-amber-500" : "bg-red-500/10 text-red-500"}
            />
            <SummaryCard
              icon={Zap}
              label="总吞吐量"
              value={formatNumber(summary.totalRPS)}
              subValue="req/s"
              color="bg-violet-500/10 text-violet-500"
            />
            <SummaryCard
              icon={AlertTriangle}
              label="告警中"
              value={summary.warningCount.toString()}
              subValue="需要关注"
              color="bg-amber-500/10 text-amber-500"
            />
            <SummaryCard
              icon={AlertTriangle}
              label="严重问题"
              value={summary.criticalCount.toString()}
              subValue="需立即处理"
              color="bg-red-500/10 text-red-500"
            />
          </div>

          {/* 域名 SLO 列表 */}
          <div>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-semibold text-default">
                域名 SLO 状态
                <span className="ml-2 text-xs font-normal text-muted">({summary.totalDomains} 个域名)</span>
              </h2>
              <button className="text-xs text-primary hover:underline">+ 添加 SLO 目标</button>
            </div>
            <div className="space-y-3">
              {domains.map((domain) => (
                <DomainCard
                  key={domain.id}
                  domain={domain}
                  expanded={expandedId === domain.id}
                  onToggle={() => setExpandedId(expandedId === domain.id ? null : domain.id)}
                  timeRange={timeRange}
                  onEditTargets={() => setShowConfigModal(domain)}
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
                  系统采集 <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded">nginx_ingress_controller_requests</code>、
                  <code className="px-1 py-0.5 bg-blue-100 dark:bg-blue-900 rounded ml-1">nginx_ingress_controller_request_duration_seconds</code> 等指标，
                  计算可用性、延迟百分位数和错误率。
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* 配置弹窗 */}
      {showConfigModal && (
        <SLOConfigModal
          domain={showConfigModal}
          onClose={() => setShowConfigModal(null)}
          onSave={(range, newTargets) => handleSaveConfig(showConfigModal.id, range, newTargets)}
          currentTimeRange={timeRange}
        />
      )}

      {/* 导出报告弹窗 */}
      {showExportModal && (
        <ExportReportModal onClose={() => setShowExportModal(false)} />
      )}
    </Layout>
  );
}
