"use client";

import { BellOff, BellRing, GitBranch, Send, SlidersHorizontal, Users, type LucideIcon } from "lucide-react";
import type { NotificationsTranslations } from "@/types/i18n";

export type NotifyTabKey = keyof NotificationsTranslations["tabs"];

/** 页签的顺序：先看现状（告警、送达），再看配置（接收人、规则、静默、策略） */
const TABS: { key: NotifyTabKey; icon: LucideIcon }[] = [
  { key: "alerts", icon: BellRing },
  { key: "deliveries", icon: Send },
  { key: "receivers", icon: Users },
  { key: "rules", icon: GitBranch },
  { key: "silences", icon: BellOff },
  { key: "policy", icon: SlidersHorizontal },
];

interface NotifyTabBarProps {
  activeTab: NotifyTabKey;
  onSelect: (tab: NotifyTabKey) => void;
  labels: NotificationsTranslations["tabs"];
}

/** 通知设置页的六个页签。窄屏时横向滚动，不折行 */
export function NotifyTabBar({ activeTab, onSelect, labels }: NotifyTabBarProps) {
  return (
    <div role="tablist" className="flex gap-1 overflow-x-auto border-b border-[var(--border-color)]">
      {TABS.map(({ key, icon: Icon }) => {
        const isActive = key === activeTab;
        return (
          <button
            key={key}
            type="button"
            role="tab"
            aria-selected={isActive}
            onClick={() => onSelect(key)}
            className={`flex items-center gap-1.5 px-3 sm:px-4 py-2.5 text-sm whitespace-nowrap border-b-2 -mb-px transition-colors ${
              isActive
                ? "border-primary text-primary font-medium"
                : "border-transparent text-muted hover:text-default"
            }`}
          >
            <Icon className="w-4 h-4" />
            {labels[key]}
          </button>
        );
      })}
    </div>
  );
}
