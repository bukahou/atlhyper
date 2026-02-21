"use client";

import { useState, useCallback, KeyboardEvent } from "react";
import { X, Plus } from "lucide-react";

interface TagInputProps {
  value: string[];
  onChange: (value: string[]) => void;
  placeholder?: string;
  disabled?: boolean;
  validator?: (value: string) => boolean;
}

/**
 * 标签输入组件
 * 用于管理邮件收件人列表
 */
export function TagInput({
  value,
  onChange,
  placeholder = "输入后按 Enter 添加",
  disabled = false,
  validator,
}: TagInputProps) {
  const [inputValue, setInputValue] = useState("");
  const [error, setError] = useState("");

  const addTag = useCallback(() => {
    const trimmed = inputValue.trim();
    if (!trimmed) return;

    // 检查重复
    if (value.includes(trimmed)) {
      setError("已存在");
      return;
    }

    // 验证格式（如邮箱）
    if (validator && !validator(trimmed)) {
      setError("格式无效");
      return;
    }

    onChange([...value, trimmed]);
    setInputValue("");
    setError("");
  }, [inputValue, value, onChange, validator]);

  const removeTag = useCallback(
    (tagToRemove: string) => {
      onChange(value.filter((tag) => tag !== tagToRemove));
    },
    [value, onChange]
  );

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      addTag();
    } else if (e.key === "Backspace" && !inputValue && value.length > 0) {
      // 删除最后一个标签
      removeTag(value[value.length - 1]);
    }
  };

  return (
    <div className="space-y-2">
      {/* 标签列表 */}
      <div className="flex flex-wrap gap-2">
        {value.map((tag) => (
          <span
            key={tag}
            className="inline-flex items-center gap-1 px-2.5 py-1 rounded-full text-sm
              bg-blue-100 dark:bg-blue-900/40 text-blue-800 dark:text-blue-300
              border border-blue-200 dark:border-blue-700"
          >
            {tag}
            {!disabled && (
              <button
                type="button"
                onClick={() => removeTag(tag)}
                className="p-0.5 hover:bg-blue-200 dark:hover:bg-blue-800 rounded-full transition-colors"
              >
                <X className="w-3 h-3" />
              </button>
            )}
          </span>
        ))}
      </div>

      {/* 输入框 */}
      {!disabled && (
        <div className="flex gap-2">
          <div className="flex-1 relative">
            <input
              type="text"
              value={inputValue}
              onChange={(e) => {
                setInputValue(e.target.value);
                setError("");
              }}
              onKeyDown={handleKeyDown}
              placeholder={placeholder}
              className={`w-full px-3 py-2 rounded-lg border text-sm
                bg-[var(--bg-primary)] text-default
                focus:outline-none focus:ring-2 focus:ring-blue-500/50
                ${error ? "border-red-500" : "border-[var(--border-color)]"}`}
            />
            {error && (
              <p className="absolute -bottom-5 left-0 text-xs text-red-500">
                {error}
              </p>
            )}
          </div>
          <button
            type="button"
            onClick={addTag}
            disabled={!inputValue.trim()}
            className="px-3 py-2 rounded-lg border border-[var(--border-color)]
              bg-[var(--bg-primary)] text-default
              hover:bg-[var(--bg-secondary)] disabled:opacity-50 disabled:cursor-not-allowed
              transition-colors"
          >
            <Plus className="w-4 h-4" />
          </button>
        </div>
      )}
    </div>
  );
}

/**
 * 邮箱验证器
 */
export function emailValidator(email: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}
