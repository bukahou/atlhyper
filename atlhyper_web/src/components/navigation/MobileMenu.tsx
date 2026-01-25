"use client";

import { X } from "lucide-react";
import { Sidebar } from "./Sidebar";

interface MobileMenuProps {
  open: boolean;
  onClose: () => void;
}

export function MobileMenu({ open, onClose }: MobileMenuProps) {
  if (!open) return null;

  return (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-40 lg:hidden"
        onClick={onClose}
      />

      {/* Menu */}
      <div className="fixed inset-y-0 left-0 z-50 w-64 bg-[var(--sidebar-bg)] lg:hidden">
        <div className="absolute top-4 right-4">
          <button
            onClick={onClose}
            className="p-2 rounded-lg hover-bg"
          >
            <X className="w-5 h-5 text-secondary" />
          </button>
        </div>
        <Sidebar collapsed={false} onToggle={onClose} />
      </div>
    </>
  );
}
