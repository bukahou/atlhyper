"use client";

import { SquarePen, MessageSquare, Trash2, EllipsisVertical } from "lucide-react";
import { useState } from "react";
import { Conversation } from "./types";

interface ConversationPanelProps {
  open: boolean;
  onClose: () => void;
  conversations: Conversation[];
  currentId: number | null;
  onSelect: (id: number) => void;
  onNew: () => void;
  onDelete: (id: number) => void;
}

// 按时间分组
function groupByTime(conversations: Conversation[]) {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const yesterday = new Date(today.getTime() - 86400000);
  const prev7 = new Date(today.getTime() - 7 * 86400000);
  const prev30 = new Date(today.getTime() - 30 * 86400000);

  const groups: { label: string; items: Conversation[] }[] = [
    { label: "今天", items: [] },
    { label: "昨天", items: [] },
    { label: "过去 7 天", items: [] },
    { label: "过去 30 天", items: [] },
    { label: "更早", items: [] },
  ];

  for (const conv of conversations) {
    const d = new Date(conv.updated_at || conv.created_at);
    if (d >= today) groups[0].items.push(conv);
    else if (d >= yesterday) groups[1].items.push(conv);
    else if (d >= prev7) groups[2].items.push(conv);
    else if (d >= prev30) groups[3].items.push(conv);
    else groups[4].items.push(conv);
  }

  return groups.filter((g) => g.items.length > 0);
}

// 对话列表面板内容
export function ConversationPanel({
  onClose,
  conversations,
  currentId,
  onSelect,
  onNew,
  onDelete,
}: ConversationPanelProps) {
  const [menuOpenId, setMenuOpenId] = useState<number | null>(null);
  const groups = groupByTime(conversations);

  return (
    <div className="flex flex-col overflow-hidden">
      {/* 头部 */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-[var(--border-color)]/50">
        <span className="text-sm font-medium text-default">对话记录</span>
        <button
          onClick={() => {
            onNew();
            onClose();
          }}
          className="flex items-center gap-1 px-2 py-1 rounded-md text-xs text-primary hover:bg-primary/10 transition-colors"
        >
          <SquarePen className="w-3.5 h-3.5" />
          新对话
        </button>
      </div>

      {/* 列表 */}
      <div className="flex-1 overflow-y-auto py-2 px-2">
        {conversations.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-muted">
            <MessageSquare className="w-8 h-8 mb-2 opacity-20" />
            <span className="text-xs">暂无对话记录</span>
          </div>
        ) : (
          groups.map((group) => (
            <div key={group.label} className="mb-2">
              <div className="px-2 py-1">
                <span className="text-[11px] font-medium text-muted">{group.label}</span>
              </div>
              {group.items.map((conv) => (
                <ConversationItem
                  key={conv.id}
                  conversation={conv}
                  active={currentId === conv.id}
                  menuOpen={menuOpenId === conv.id}
                  onSelect={() => {
                    onSelect(conv.id);
                    onClose();
                  }}
                  onDelete={() => {
                    onDelete(conv.id);
                    setMenuOpenId(null);
                  }}
                  onMenuToggle={() =>
                    setMenuOpenId((prev) => (prev === conv.id ? null : conv.id))
                  }
                />
              ))}
            </div>
          ))
        )}
      </div>
    </div>
  );
}

// 单个对话条目
function ConversationItem({
  conversation,
  active,
  menuOpen,
  onSelect,
  onDelete,
  onMenuToggle,
}: {
  conversation: Conversation;
  active: boolean;
  menuOpen: boolean;
  onSelect: () => void;
  onDelete: () => void;
  onMenuToggle: () => void;
}) {
  return (
    <div className="relative">
      <div
        onClick={onSelect}
        className={`group flex items-center gap-2 px-3 py-2 rounded-lg cursor-pointer transition-colors text-sm ${
          active
            ? "bg-primary/10 text-primary font-medium"
            : "text-secondary hover:bg-[var(--hover-bg)] hover:text-default"
        }`}
      >
        <MessageSquare className="w-3.5 h-3.5 flex-shrink-0 opacity-50" />
        <span className="flex-1 truncate">{conversation.title}</span>
        <button
          onClick={(e) => {
            e.stopPropagation();
            onMenuToggle();
          }}
          className={`p-1 rounded transition-opacity ${
            menuOpen ? "opacity-100" : "opacity-0 group-hover:opacity-100"
          } hover:bg-[var(--border-color)]`}
        >
          <EllipsisVertical className="w-3.5 h-3.5 text-muted" />
        </button>
      </div>

      {/* 下拉菜单 */}
      {menuOpen && (
        <div className="absolute right-2 top-9 z-40 w-28 py-1 rounded-lg border border-[var(--border-color)] bg-card shadow-lg">
          <button
            onClick={onDelete}
            className="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20 transition-colors"
          >
            <Trash2 className="w-3 h-3" />
            删除对话
          </button>
        </div>
      )}
    </div>
  );
}
