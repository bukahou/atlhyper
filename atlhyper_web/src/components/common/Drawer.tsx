"use client";

import { useEffect } from "react";
import { X } from "lucide-react";

interface DrawerProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  size?: "sm" | "md" | "lg" | "xl" | "full";
}

const sizeClasses: Record<string, string> = {
  sm: "w-[400px]",
  md: "w-[520px]",
  lg: "w-[640px]",
  xl: "w-[720px]",
  full: "w-[90vw]",
};

export function Drawer({ isOpen, onClose, title, children, size = "lg" }: DrawerProps) {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    if (isOpen) {
      document.addEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "hidden";
    }
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.body.style.overflow = "";
    };
  }, [isOpen, onClose]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/50 z-50 flex justify-end" onClick={onClose}>
      <div
        className={`bg-card shadow-2xl max-w-full ${sizeClasses[size]} h-full flex flex-col animate-slide-in-right`}
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-[var(--border-color)] px-6 py-4 shrink-0">
          <h2 className="text-lg font-semibold text-default truncate">{title}</h2>
          <button onClick={onClose} className="p-2 rounded-lg hover-bg shrink-0">
            <X className="w-5 h-5 text-muted" />
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto min-h-0">{children}</div>
      </div>
    </div>
  );
}
