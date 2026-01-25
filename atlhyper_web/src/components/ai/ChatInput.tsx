"use client";

import { useState, useRef, useEffect } from "react";
import { ArrowUp, Square } from "lucide-react";

interface ChatInputProps {
  onSend: (message: string) => void;
  onStop?: () => void;
  disabled?: boolean;
  streaming?: boolean;
}

export function ChatInput({ onSend, onStop, disabled, streaming }: ChatInputProps) {
  const [input, setInput] = useState("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  // 自动调整高度
  useEffect(() => {
    const el = textareaRef.current;
    if (el) {
      el.style.height = "auto";
      el.style.height = Math.min(el.scrollHeight, 200) + "px";
    }
  }, [input]);

  const handleSend = () => {
    const trimmed = input.trim();
    if (!trimmed || disabled) return;
    onSend(trimmed);
    setInput("");
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  return (
    <div className="px-4 pb-4 pt-2">
      <div className="relative max-w-3xl mx-auto">
        <div className="flex items-end rounded-2xl border border-[var(--border-color)] bg-[var(--background)] shadow-sm focus-within:border-[var(--border-color)] focus-within:shadow-md transition-shadow">
          <textarea
            ref={textareaRef}
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={streaming ? "AI 正在回复..." : "给 AI 助手发送消息..."}
            disabled={disabled || streaming}
            rows={1}
            className="flex-1 resize-none bg-transparent pl-4 pr-2 py-3 text-sm text-default placeholder:text-muted focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed min-h-[44px] max-h-[200px]"
          />
          <div className="flex items-center pr-2 pb-2">
            {streaming ? (
              <button
                onClick={onStop}
                className="w-8 h-8 flex items-center justify-center rounded-lg bg-default text-white hover:opacity-80 transition-opacity"
                title="停止生成"
              >
                <Square className="w-3.5 h-3.5" fill="currentColor" />
              </button>
            ) : (
              <button
                onClick={handleSend}
                disabled={!input.trim() || disabled}
                className="w-8 h-8 flex items-center justify-center rounded-lg bg-primary text-white transition-opacity disabled:opacity-30 disabled:cursor-not-allowed hover:opacity-90"
                title="发送"
              >
                <ArrowUp className="w-4 h-4" strokeWidth={2.5} />
              </button>
            )}
          </div>
        </div>
        <p className="text-center text-[11px] text-muted mt-2">
          AI 可能会出错，请核实重要信息。
        </p>
      </div>
    </div>
  );
}
