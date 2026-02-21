"use client";

import { useState, useEffect, useMemo } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { Database, Layers } from "lucide-react";
import {
  MODULE_REGISTRY,
  getDataSourceMode,
  setDataSourceMode,
} from "@/config/data-source";
import type { DataSourceMode, ModuleDataSource } from "@/config/data-source";

// 分组顺序
const CATEGORY_ORDER = ["observe", "cluster", "admin", "settings", "aiops"] as const;

function groupLabel(category: string, t: ReturnType<typeof useI18n>["t"]["dataSource"]): string {
  const map: Record<string, string> = {
    observe: t.groupObserve,
    cluster: t.groupCluster,
    admin: t.groupAdmin,
    settings: t.groupSettings,
    aiops: t.groupAiops,
  };
  return map[category] ?? category;
}

/** 内联 Toggle 开关 */
function Toggle({
  checked,
  onChange,
  disabled,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => !disabled && onChange(!checked)}
      className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
        disabled
          ? "bg-[var(--border-color)] cursor-not-allowed opacity-50"
          : checked
            ? "bg-primary"
            : "bg-[var(--border-color)]"
      }`}
    >
      <span
        className={`inline-block h-3.5 w-3.5 rounded-full bg-white shadow transition-transform ${
          checked ? "translate-x-[18px]" : "translate-x-[3px]"
        }`}
      />
    </button>
  );
}

/** Toast 通知 */
function Toast({ message, onClose }: { message: string; onClose: () => void }) {
  useEffect(() => {
    const timer = setTimeout(onClose, 3000);
    return () => clearTimeout(timer);
  }, [onClose]);

  return (
    <div className="fixed bottom-6 right-6 z-50 animate-in fade-in slide-in-from-bottom-4 duration-300">
      <div className="px-4 py-2.5 rounded-lg bg-card border border-[var(--border-color)] shadow-lg text-sm text-default">
        {message}
      </div>
    </div>
  );
}

/** 模块行 */
function ModuleRow({
  mod,
  mode,
  navLabel,
  tds,
  onToggle,
}: {
  mod: ModuleDataSource;
  mode: DataSourceMode;
  navLabel: string;
  tds: ReturnType<typeof useI18n>["t"]["dataSource"];
  onToggle: (key: string, mode: DataSourceMode) => void;
}) {
  const isMock = mode === "mock";

  return (
    <div className="flex items-center justify-between py-2.5 px-3 rounded-lg hover:bg-[var(--hover-bg)] transition-colors">
      <div className="flex items-center gap-3">
        <span className="text-sm text-default">{navLabel}</span>
        {mod.hasMock ? (
          <span
            className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${
              isMock
                ? "bg-amber-500/10 text-amber-600 dark:text-amber-400"
                : "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400"
            }`}
          >
            {isMock ? tds.mock : tds.api}
          </span>
        ) : (
          <span className="text-[10px] px-1.5 py-0.5 rounded bg-gray-500/10 text-muted font-medium">
            {tds.apiOnly}
          </span>
        )}
      </div>
      <div className="flex items-center gap-2">
        {mod.hasMock && (
          <>
            <span className="text-[10px] text-muted">{tds.mock}</span>
            <Toggle
              checked={!isMock}
              onChange={(toApi) =>
                onToggle(mod.key, toApi ? "api" : "mock")
              }
            />
            <span className="text-[10px] text-muted">{tds.api}</span>
          </>
        )}
        {!mod.hasMock && (
          <Toggle checked={true} onChange={() => {}} disabled />
        )}
      </div>
    </div>
  );
}

export default function DataSourcePage() {
  const { t } = useI18n();
  const tds = t.dataSource;
  const tn = t.nav;

  // 从 localStorage 加载当前配置
  const [modes, setModes] = useState<Record<string, DataSourceMode>>({});
  const [toast, setToast] = useState<string | null>(null);

  useEffect(() => {
    const initial: Record<string, DataSourceMode> = {};
    for (const mod of MODULE_REGISTRY) {
      initial[mod.key] = getDataSourceMode(mod.key);
    }
    setModes(initial);
  }, []);

  const handleToggle = (key: string, newMode: DataSourceMode) => {
    setDataSourceMode(key, newMode);
    setModes((prev) => ({ ...prev, [key]: newMode }));
    const msg =
      newMode === "mock" ? tds.switchedToMock : tds.switchedToApi;
    setToast(`${msg} — ${tds.refreshHint}`);
  };

  // 按分组
  const grouped = useMemo(() => {
    const map = new Map<string, ModuleDataSource[]>();
    for (const mod of MODULE_REGISTRY) {
      const list = map.get(mod.category) || [];
      list.push(mod);
      map.set(mod.category, list);
    }
    return map;
  }, []);

  // 统计
  const stats = useMemo(() => {
    const total = MODULE_REGISTRY.length;
    let mockCount = 0;
    for (const mod of MODULE_REGISTRY) {
      if (modes[mod.key] === "mock") mockCount++;
    }
    return { total, mockCount, apiCount: total - mockCount };
  }, [modes]);

  return (
    <Layout>
      <div className="space-y-6">
        {/* 标题 */}
        <div>
          <h1 className="text-xl font-bold text-default">{tds.pageTitle}</h1>
          <p className="text-xs text-muted mt-1">{tds.pageDescription}</p>
        </div>

        {/* 统计条 */}
        <div className="grid grid-cols-3 gap-3">
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 text-center">
            <div className="text-lg font-bold text-default">{stats.total}</div>
            <div className="text-[10px] text-muted">{tds.totalModules}</div>
          </div>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 text-center">
            <div className="text-lg font-bold text-amber-500">{stats.mockCount}</div>
            <div className="text-[10px] text-muted">{tds.mockCount}</div>
          </div>
          <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 text-center">
            <div className="text-lg font-bold text-emerald-500">{stats.apiCount}</div>
            <div className="text-[10px] text-muted">{tds.apiCount}</div>
          </div>
        </div>

        {/* 分组卡片 */}
        {CATEGORY_ORDER.map((cat) => {
          const modules = grouped.get(cat);
          if (!modules) return null;

          return (
            <div
              key={cat}
              className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden"
            >
              {/* 分组标题 */}
              <div className="flex items-center gap-2 px-4 py-3 border-b border-[var(--border-color)]/50">
                {cat === "observe" ? (
                  <Layers className="w-4 h-4 text-primary" />
                ) : (
                  <Database className="w-4 h-4 text-primary" />
                )}
                <span className="text-sm font-semibold text-default">
                  {groupLabel(cat, tds)}
                </span>
                <span className="text-[10px] text-muted">
                  ({modules.length})
                </span>
              </div>

              {/* 模块列表 */}
              <div className="px-1 py-1 divide-y divide-[var(--border-color)]/30">
                {modules.map((mod) => (
                  <ModuleRow
                    key={mod.key}
                    mod={mod}
                    mode={modes[mod.key] ?? mod.defaultMode}
                    navLabel={tn[mod.labelKey as keyof typeof tn] ?? mod.key}
                    tds={tds}
                    onToggle={handleToggle}
                  />
                ))}
              </div>
            </div>
          );
        })}
      </div>

      {/* Toast */}
      {toast && <Toast message={toast} onClose={() => setToast(null)} />}
    </Layout>
  );
}
