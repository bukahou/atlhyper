"use client";

/**
 * 演示数据横幅
 *
 * 纯管理功能页在当前主体无权限时整页展示演示数据，页首必须用它醒目标注。
 * ⛔ 真实数据与演示数据不得出现在同一视图。
 */

import { Eye } from "lucide-react";
import { useI18n } from "@/i18n/context";

export function DemoBanner({ hint }: { hint: string }) {
  const { t } = useI18n();
  return (
    <div className="flex items-center gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 rounded-xl">
      <Eye className="w-5 h-5 text-amber-600 dark:text-amber-400 flex-shrink-0" />
      <div>
        <p className="text-sm font-medium text-amber-800 dark:text-amber-300">{t.common.demoMode}</p>
        <p className="text-xs text-amber-600 dark:text-amber-400">{hint}</p>
      </div>
    </div>
  );
}
