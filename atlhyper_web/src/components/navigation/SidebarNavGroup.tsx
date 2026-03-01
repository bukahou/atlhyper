"use client";

import Link from "next/link";
import { ChevronDown, ChevronRight } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { NavGroup } from "./SidebarTypes";

interface SidebarNavGroupProps {
  group: NavGroup;
  collapsed: boolean;
  isAdmin: boolean;
  isActive: (href: string) => boolean;
  isGroupActive: boolean;
  isExpanded: boolean;
  hoveredGroup: string | null;
  onToggleGroup: (key: string) => void;
  onHoverGroup: (key: string | null) => void;
}

export function SidebarNavGroup({
  group,
  collapsed,
  isAdmin,
  isActive,
  isGroupActive,
  isExpanded,
  hoveredGroup,
  onToggleGroup,
  onHoverGroup,
}: SidebarNavGroupProps) {
  const { t } = useI18n();
  const Icon = group.icon;
  const hasChildren = !!group.children;

  // 折叠模式: icon-only + hover flyout
  if (collapsed) {
    return (
      <div
        className="relative mb-0.5"
        onMouseEnter={() => hasChildren && onHoverGroup(group.key)}
        onMouseLeave={() => onHoverGroup(null)}
      >
        {group.href ? (
          <Link
            href={group.href}
            className={`flex items-center justify-center w-10 h-10 mx-auto rounded-xl transition-all duration-150 ${
              isGroupActive ? "bg-primary/15 text-primary shadow-sm" : "text-muted hover:bg-white/5 dark:hover:bg-white/5 hover:text-default"
            }`}
            title={t.nav[group.key as keyof typeof t.nav]}
          >
            <Icon className="w-5 h-5" />
          </Link>
        ) : (
          <button
            className={`flex items-center justify-center w-10 h-10 mx-auto rounded-xl transition-all duration-150 ${
              isGroupActive ? "bg-primary/15 text-primary shadow-sm" : "text-muted hover:bg-white/5 dark:hover:bg-white/5 hover:text-default"
            }`}
            title={t.nav[group.key as keyof typeof t.nav]}
          >
            <Icon className="w-5 h-5" />
          </button>
        )}

        {/* Flyout (collapsed mode) */}
        {hasChildren && hoveredGroup === group.key && (
          <div className="absolute left-full top-0 pl-2 z-50">
            <div className="py-3 px-2 min-w-[180px] rounded-2xl border border-white/10 dark:border-white/5 bg-card/95 backdrop-blur-xl shadow-[0_8px_30px_rgb(0,0,0,0.12)] dark:shadow-[0_8px_30px_rgb(0,0,0,0.4)] ring-1 ring-black/5 dark:ring-white/10 animate-in fade-in slide-in-from-left-2 duration-200">
              <div className="px-3 py-2 text-[11px] font-semibold text-muted uppercase tracking-wider border-b border-[var(--border-color)]/30 mb-1">
                {t.nav[group.key as keyof typeof t.nav]}
              </div>
              {group.children!
                .filter((child) => !child.adminOnly || isAdmin)
                .map((child, idx) => {
                  const ChildIcon = child.icon;
                  return (
                    <div key={child.key}>
                      {child.section && idx > 0 && (
                        <div className="flex items-center gap-2 px-3 pt-2 pb-1">
                          <div className="flex-1 h-px bg-[var(--border-color)]/30" />
                          <span className="text-[10px] text-muted uppercase tracking-wider font-medium">
                            {t.nav[child.section as keyof typeof t.nav]}
                          </span>
                          <div className="flex-1 h-px bg-[var(--border-color)]/30" />
                        </div>
                      )}
                      <Link
                        href={child.href}
                        className={`flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm whitespace-nowrap transition-all duration-150 ${
                          isActive(child.href)
                            ? "bg-primary/15 text-primary font-medium shadow-sm"
                            : "text-secondary hover:bg-[var(--hover-bg)] hover:text-default hover:translate-x-0.5"
                        }`}
                      >
                        <ChildIcon className="w-4 h-4" />
                        {t.nav[child.key as keyof typeof t.nav]}
                      </Link>
                    </div>
                  );
                })}
            </div>
          </div>
        )}
      </div>
    );
  }

  // 展开模式: 完整导航 (有子项的组)
  if (hasChildren) {
    return (
      <div className="mb-1.5">
        <button
          onClick={() => onToggleGroup(group.key)}
          className={`w-full flex items-center gap-3 px-3 py-2.5 rounded-xl transition-all duration-150 text-sm ${
            isGroupActive ? "text-primary bg-primary/5" : "text-secondary hover:bg-white/5 dark:hover:bg-white/5"
          }`}
        >
          <Icon className="w-[18px] h-[18px] flex-shrink-0" />
          <span className="flex-1 text-left font-medium">{t.nav[group.key as keyof typeof t.nav]}</span>
          {isExpanded ? <ChevronDown className="w-3.5 h-3.5 text-muted transition-transform" /> : <ChevronRight className="w-3.5 h-3.5 text-muted transition-transform" />}
        </button>
        {isExpanded && (
          <div className="mt-1 ml-2 pl-4 border-l-2 border-primary/20">
            {group.children!
              .filter((child) => !child.adminOnly || isAdmin)
              .map((child, idx) => {
                const ChildIcon = child.icon;
                return (
                  <div key={child.key}>
                    {child.section && idx > 0 && (
                      <div className="flex items-center gap-2 px-3 pt-2.5 pb-1">
                        <div className="flex-1 h-px bg-[var(--border-color)]/30" />
                        <span className="text-[10px] text-muted uppercase tracking-wider font-medium">
                          {t.nav[child.section as keyof typeof t.nav]}
                        </span>
                        <div className="flex-1 h-px bg-[var(--border-color)]/30" />
                      </div>
                    )}
                    <Link
                      href={child.href}
                      className={`flex items-center gap-2.5 px-3 py-2 rounded-xl text-sm transition-all duration-150 ${
                        isActive(child.href)
                          ? "bg-primary/15 text-primary font-medium shadow-sm"
                          : "text-muted hover:bg-white/5 dark:hover:bg-white/5 hover:text-default hover:translate-x-0.5"
                      }`}
                    >
                      <ChildIcon className="w-4 h-4" />
                      {t.nav[child.key as keyof typeof t.nav]}
                    </Link>
                  </div>
                );
              })}
          </div>
        )}
      </div>
    );
  }

  // 展开模式: 无子项的单独链接
  return (
    <Link
      href={group.href!}
      className={`flex items-center gap-3 px-3 py-2.5 rounded-xl text-sm transition-all duration-150 mb-1.5 ${
        isGroupActive ? "bg-primary/15 text-primary font-medium shadow-sm" : "text-secondary hover:bg-white/5 dark:hover:bg-white/5"
      }`}
    >
      <Icon className="w-[18px] h-[18px] flex-shrink-0" />
      <span className="font-medium">{t.nav[group.key as keyof typeof t.nav]}</span>
    </Link>
  );
}
