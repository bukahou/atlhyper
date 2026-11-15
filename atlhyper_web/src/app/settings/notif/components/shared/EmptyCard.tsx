"use client";

import type { LucideIcon } from "lucide-react";

interface EmptyCardProps {
  icon: LucideIcon;
  title: string;
  hint?: string;
}

/** 列表为空时的占位：组件不隐藏，说明为什么是空的 */
export function EmptyCard({ icon: Icon, title, hint }: EmptyCardProps) {
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] py-12 px-4 text-center">
      <Icon className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
      <p className="text-default font-medium mb-1">{title}</p>
      {hint && <p className="text-sm text-muted">{hint}</p>}
    </div>
  );
}
