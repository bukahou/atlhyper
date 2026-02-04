"use client";

import { useState, useEffect, useRef, useCallback } from "react";
import { Modal } from "@/components/common/Modal";
import { LoadingSpinner } from "@/components/common/LoadingSpinner";
import { getPodLogs } from "@/api/pod";
import { getCurrentClusterId } from "@/config/cluster";
import {
  RefreshCw,
  Download,
  ArrowDown,
  Search,
  X,
  ChevronDown,
} from "lucide-react";

interface PodLogsViewerProps {
  isOpen: boolean;
  onClose: () => void;
  namespace: string;
  podName: string;
  containerName: string;
}

export function PodLogsViewer({
  isOpen,
  onClose,
  namespace,
  podName,
  containerName,
}: PodLogsViewerProps) {
  const [logs, setLogs] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [tailLines, setTailLines] = useState(100);
  const [autoScroll, setAutoScroll] = useState(true);
  const [searchText, setSearchText] = useState("");
  const [showSearch, setShowSearch] = useState(false);
  const logsRef = useRef<HTMLPreElement>(null);

  const fetchLogs = useCallback(async () => {
    if (!namespace || !podName) return;
    setLoading(true);
    setError("");
    try {
      const res = await getPodLogs({
        ClusterID: getCurrentClusterId(),
        Namespace: namespace,
        Pod: podName,
        Container: containerName || undefined,
        TailLines: tailLines,
        TimeoutSeconds: 30,
      });
      setLogs(res.data.data?.logs || "暂无日志");
    } catch (err) {
      setError(err instanceof Error ? err.message : "获取日志失败");
    } finally {
      setLoading(false);
    }
  }, [namespace, podName, containerName, tailLines]);

  useEffect(() => {
    if (isOpen) {
      fetchLogs();
    }
  }, [isOpen, fetchLogs]);

  // 自动滚动到底部
  useEffect(() => {
    if (autoScroll && logsRef.current && !loading) {
      logsRef.current.scrollTop = logsRef.current.scrollHeight;
    }
  }, [logs, autoScroll, loading]);

  // 下载日志
  const handleDownload = () => {
    const blob = new Blob([logs], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${podName}-${containerName || "logs"}.txt`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  // 高亮搜索文本
  const highlightLogs = (text: string) => {
    if (!searchText.trim()) return text;
    const regex = new RegExp(`(${searchText.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`, "gi");
    return text.replace(regex, '<mark class="bg-yellow-300 dark:bg-yellow-600">$1</mark>');
  };

  const tailLinesOptions = [100, 200, 500, 1000, 2000];

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={`日志: ${podName}${containerName ? ` / ${containerName}` : ""}`}
      size="full"
    >
      <div className="flex flex-col h-full">
        {/* 工具栏 */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2 px-3 sm:px-4 py-2 sm:py-3 border-b border-[var(--border-color)] bg-[var(--background)] shrink-0">
          {/* 左侧控制 */}
          <div className="flex items-center gap-2 sm:gap-3 flex-wrap">
            {/* 行数选择 */}
            <div className="flex items-center gap-1.5 sm:gap-2">
              <span className="text-xs sm:text-sm text-muted hidden sm:inline">显示最后</span>
              <div className="relative">
                <select
                  value={tailLines}
                  onChange={(e) => setTailLines(Number(e.target.value))}
                  className="appearance-none pl-2 sm:pl-3 pr-6 sm:pr-8 py-1.5 bg-card border border-[var(--border-color)] rounded text-xs sm:text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary"
                >
                  {tailLinesOptions.map((n) => (
                    <option key={n} value={n}>
                      {n}
                    </option>
                  ))}
                </select>
                <ChevronDown className="absolute right-1.5 sm:right-2 top-1/2 -translate-y-1/2 w-3 h-3 sm:w-4 sm:h-4 text-muted pointer-events-none" />
              </div>
              <span className="text-xs sm:text-sm text-muted">行</span>
            </div>

            {/* 搜索按钮 */}
            <button
              onClick={() => setShowSearch(!showSearch)}
              className={`p-1.5 sm:p-2 rounded-lg transition-colors ${
                showSearch ? "bg-primary/10 text-primary" : "hover-bg text-muted"
              }`}
              title="搜索"
            >
              <Search className="w-4 h-4" />
            </button>

            {/* 搜索框 - 移动端显示在下方 */}
            {showSearch && (
              <div className="relative w-full sm:w-auto order-last sm:order-none">
                <input
                  type="text"
                  value={searchText}
                  onChange={(e) => setSearchText(e.target.value)}
                  placeholder="搜索日志..."
                  className="w-full sm:w-48 pl-3 pr-8 py-1.5 bg-card border border-[var(--border-color)] rounded text-sm text-default focus:outline-none focus:ring-1 focus:ring-primary"
                  autoFocus
                />
                {searchText && (
                  <button
                    onClick={() => setSearchText("")}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-muted hover:text-default p-1"
                  >
                    <X className="w-3 h-3" />
                  </button>
                )}
              </div>
            )}
          </div>

          {/* 右侧按钮 */}
          <div className="flex items-center gap-1 sm:gap-2">
            {/* 自动滚动 */}
            <button
              onClick={() => setAutoScroll(!autoScroll)}
              className={`flex items-center gap-1 sm:gap-1.5 px-2 sm:px-3 py-1.5 rounded text-xs sm:text-sm transition-colors ${
                autoScroll
                  ? "bg-primary/10 text-primary"
                  : "hover-bg text-muted"
              }`}
              title="自动滚动到底部"
            >
              <ArrowDown className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
              <span className="hidden sm:inline">自动滚动</span>
            </button>

            {/* 刷新 */}
            <button
              onClick={fetchLogs}
              disabled={loading}
              className="p-1.5 sm:p-2 rounded-lg hover-bg text-muted hover:text-default disabled:opacity-50 transition-colors"
              title="刷新"
            >
              <RefreshCw className={`w-4 h-4 ${loading ? "animate-spin" : ""}`} />
            </button>

            {/* 下载 */}
            <button
              onClick={handleDownload}
              disabled={!logs || loading}
              className="p-1.5 sm:p-2 rounded-lg hover-bg text-muted hover:text-default disabled:opacity-50 transition-colors"
              title="下载日志"
            >
              <Download className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* 日志内容 */}
        <div className="flex-1 overflow-hidden min-h-0">
          {loading && !logs ? (
            <div className="h-full flex items-center justify-center">
              <LoadingSpinner />
            </div>
          ) : error ? (
            <div className="h-full flex items-center justify-center text-red-500 p-4 text-sm">{error}</div>
          ) : (
            <pre
              ref={logsRef}
              className="h-full overflow-auto p-2 sm:p-4 bg-gray-900 text-gray-100 text-[10px] sm:text-xs font-mono leading-relaxed whitespace-pre-wrap break-all"
              dangerouslySetInnerHTML={{
                __html: searchText ? highlightLogs(logs) : logs,
              }}
            />
          )}
        </div>

        {/* 状态栏 */}
        <div className="flex items-center justify-between px-3 sm:px-4 py-1.5 sm:py-2 border-t border-[var(--border-color)] bg-[var(--background)] text-[10px] sm:text-xs text-muted shrink-0">
          <span>
            {logs ? `${logs.split("\n").length} 行` : "暂无日志"}
          </span>
          {loading && <span className="text-primary">加载中...</span>}
        </div>
      </div>
    </Modal>
  );
}
