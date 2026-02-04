"use client";

import { X } from "lucide-react";

interface FilterInputProps {
  value: string;
  onChange: (value: string) => void;
  onClear: () => void;
  placeholder: string;
}

export function FilterInput({ value, onChange, onClear, placeholder }: FilterInputProps) {
  return (
    <div className="relative flex-1 min-w-[200px]">
      <input
        type="text"
        placeholder={placeholder}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="w-full px-3 py-2 pr-8 bg-[var(--background)] border border-[var(--border-color)] rounded-lg text-sm text-default placeholder:text-muted focus:outline-none focus:ring-1 focus:ring-primary"
      />
      {value && (
        <button
          onClick={onClear}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted hover:text-default transition-colors"
        >
          <X className="w-3 h-3" />
        </button>
      )}
    </div>
  );
}
