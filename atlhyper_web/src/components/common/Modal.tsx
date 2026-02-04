"use client";

import { useEffect } from "react";
import { X } from "lucide-react";

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  size?: "sm" | "md" | "lg" | "xl" | "full";
}

const sizeClasses = {
  sm: "max-w-md",
  md: "max-w-2xl",
  lg: "max-w-4xl",
  xl: "max-w-6xl",
  full: "max-w-full sm:max-w-xl md:max-w-3xl lg:max-w-5xl xl:max-w-6xl",
};

export function Modal({ isOpen, onClose, title, children, size = "lg" }: ModalProps) {
  // 按 Escape 键关闭
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

  const isFullSize = size === "full";

  return (
    <>
      {/* Backdrop + Dialog Container */}
      <div
        className={`fixed inset-0 bg-black/50 z-50 flex ${
          isFullSize ? "items-end sm:items-center justify-center p-0 sm:p-4" : "items-center justify-center p-4"
        }`}
        onClick={onClose}
      >
        <div
          className={`bg-card shadow-2xl w-full ${sizeClasses[size]} flex flex-col ${
            isFullSize
              ? "h-[92vh] sm:h-[85vh] sm:max-h-[85vh] rounded-t-xl sm:rounded-xl"
              : "max-h-[90vh] rounded-xl"
          }`}
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className={`flex items-center justify-between border-b border-[var(--border-color)] shrink-0 ${
            isFullSize ? "px-3 sm:px-6 py-3 sm:py-4" : "px-6 py-4"
          }`}>
            <h2 className={`font-semibold text-default truncate ${
              isFullSize ? "text-base sm:text-lg" : "text-lg"
            }`}>{title}</h2>
            <button onClick={onClose} className="p-2 rounded-lg hover-bg shrink-0">
              <X className="w-5 h-5 text-muted" />
            </button>
          </div>

          {/* Content */}
          <div className="flex-1 overflow-auto min-h-0">{children}</div>
        </div>
      </div>
    </>
  );
}
