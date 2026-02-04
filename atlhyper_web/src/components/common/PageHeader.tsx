"use client";

import { RefreshCw } from "lucide-react";
import { useI18n } from "@/i18n/context";

interface PageHeaderProps {
  title: string;
  description?: string;
  /** 自动刷新间隔（秒），显示在刷新按钮旁 */
  autoRefreshSeconds?: number;
  onRefresh?: () => void;
  actions?: React.ReactNode;
}

export function PageHeader({ title, description, autoRefreshSeconds, onRefresh, actions }: PageHeaderProps) {
  const { t } = useI18n();

  return (
    <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
      <div>
        <h1 className="text-xl sm:text-2xl font-bold text-gray-900 dark:text-gray-100">
          {title}
        </h1>
        {description && (
          <p className="text-sm sm:text-base text-gray-500 dark:text-gray-400 mt-1">{description}</p>
        )}
      </div>
      <div className="flex items-center gap-2 sm:gap-3 self-end sm:self-auto">
        {actions}
        {onRefresh && (
          <div className="flex items-center gap-2">
            {autoRefreshSeconds && (
              <span className="text-xs text-muted bg-[var(--background)] px-2 py-1 rounded">
                {autoRefreshSeconds}s
              </span>
            )}
            <button
              onClick={onRefresh}
              className="flex items-center gap-2 px-3 py-2.5 sm:py-2 bg-primary hover:bg-primary-hover text-white rounded-lg transition-colors"
              title={t.common.refresh}
            >
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
