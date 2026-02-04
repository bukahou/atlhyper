"use client";

import { memo, useState } from "react";
import { ListTree, ChevronDown, ChevronUp, Search } from "lucide-react";
import type { ProcessMetrics } from "@/types/node-metrics";
import { formatBytes } from "../mock/data";

interface ProcessTableProps {
  data: ProcessMetrics[];
}

type SortKey = "cpuPercent" | "memPercent" | "memBytes" | "pid";
type SortOrder = "asc" | "desc";

const getStateColor = (state: string) => {
  switch (state) {
    case "R": return "text-green-500 bg-green-500/10";  // Running
    case "S": return "text-blue-500 bg-blue-500/10";    // Sleeping
    case "D": return "text-yellow-500 bg-yellow-500/10"; // Disk wait
    case "Z": return "text-red-500 bg-red-500/10";      // Zombie
    case "T": return "text-gray-500 bg-gray-500/10";    // Stopped
    default: return "text-muted bg-[var(--background)]";
  }
};

const getStateName = (state: string) => {
  switch (state) {
    case "R": return "Running";
    case "S": return "Sleep";
    case "D": return "D.Wait";
    case "Z": return "Zombie";
    case "T": return "Stop";
    default: return state;
  }
};

export const ProcessTable = memo(function ProcessTable({ data }: ProcessTableProps) {
  const [sortKey, setSortKey] = useState<SortKey>("cpuPercent");
  const [sortOrder, setSortOrder] = useState<SortOrder>("desc");
  const [filter, setFilter] = useState("");

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortOrder(sortOrder === "asc" ? "desc" : "asc");
    } else {
      setSortKey(key);
      setSortOrder("desc");
    }
  };

  const SortIcon = ({ columnKey }: { columnKey: SortKey }) => {
    if (sortKey !== columnKey) return null;
    return sortOrder === "desc" ? (
      <ChevronDown className="w-3 h-3" />
    ) : (
      <ChevronUp className="w-3 h-3" />
    );
  };

  const filteredData = data.filter(
    (p) =>
      p.name.toLowerCase().includes(filter.toLowerCase()) ||
      p.user.toLowerCase().includes(filter.toLowerCase()) ||
      p.pid.toString().includes(filter)
  );

  const sortedData = [...filteredData].sort((a, b) => {
    const multiplier = sortOrder === "desc" ? -1 : 1;
    return (a[sortKey] - b[sortKey]) * multiplier;
  });

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 bg-indigo-500/10 rounded-lg">
            <ListTree className="w-5 h-5 text-indigo-500" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-default">Top Processes</h3>
            <p className="text-xs text-muted">{data.length} processes</p>
          </div>
        </div>

        {/* 搜索框 */}
        <div className="relative">
          <Search className="absolute left-2 top-1/2 transform -translate-y-1/2 w-4 h-4 text-muted" />
          <input
            type="text"
            placeholder="Filter..."
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="pl-8 pr-3 py-1.5 text-sm bg-[var(--background)] border border-[var(--border-color)] rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500/50"
          />
        </div>
      </div>

      {/* 表格 */}
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-xs text-muted border-b border-[var(--border-color)]">
              <th
                className="pb-2 pr-4 cursor-pointer hover:text-default"
                onClick={() => handleSort("pid")}
              >
                <div className="flex items-center gap-1">
                  PID
                  <SortIcon columnKey="pid" />
                </div>
              </th>
              <th className="pb-2 pr-4">Name</th>
              <th className="pb-2 pr-4">User</th>
              <th className="pb-2 pr-4">State</th>
              <th
                className="pb-2 pr-4 cursor-pointer hover:text-default text-right"
                onClick={() => handleSort("cpuPercent")}
              >
                <div className="flex items-center justify-end gap-1">
                  CPU%
                  <SortIcon columnKey="cpuPercent" />
                </div>
              </th>
              <th
                className="pb-2 pr-4 cursor-pointer hover:text-default text-right"
                onClick={() => handleSort("memPercent")}
              >
                <div className="flex items-center justify-end gap-1">
                  MEM%
                  <SortIcon columnKey="memPercent" />
                </div>
              </th>
              <th
                className="pb-2 cursor-pointer hover:text-default text-right"
                onClick={() => handleSort("memBytes")}
              >
                <div className="flex items-center justify-end gap-1">
                  MEM
                  <SortIcon columnKey="memBytes" />
                </div>
              </th>
            </tr>
          </thead>
          <tbody>
            {sortedData.map((proc) => (
              <tr
                key={proc.pid}
                className="border-b border-[var(--border-color)] last:border-0 hover:bg-[var(--background)]"
              >
                <td className="py-2 pr-4 text-muted">{proc.pid}</td>
                <td className="py-2 pr-4">
                  <div className="flex flex-col">
                    <span className="font-medium text-default truncate max-w-[200px]" title={proc.name}>
                      {proc.name}
                    </span>
                    <span className="text-xs text-muted truncate max-w-[200px]" title={proc.command}>
                      {proc.command}
                    </span>
                  </div>
                </td>
                <td className="py-2 pr-4 text-muted">{proc.user}</td>
                <td className="py-2 pr-4">
                  <span className={`px-1.5 py-0.5 text-xs rounded ${getStateColor(proc.state)}`}>
                    {getStateName(proc.state)}
                  </span>
                </td>
                <td className="py-2 pr-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <div className="w-12 h-1.5 bg-[var(--background)] rounded-full overflow-hidden">
                      <div
                        className="h-full bg-orange-500 rounded-full"
                        style={{ width: `${Math.min(100, proc.cpuPercent)}%` }}
                      />
                    </div>
                    <span className={proc.cpuPercent > 50 ? "text-orange-500" : "text-default"}>
                      {proc.cpuPercent.toFixed(1)}%
                    </span>
                  </div>
                </td>
                <td className="py-2 pr-4 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <div className="w-12 h-1.5 bg-[var(--background)] rounded-full overflow-hidden">
                      <div
                        className="h-full bg-green-500 rounded-full"
                        style={{ width: `${Math.min(100, proc.memPercent)}%` }}
                      />
                    </div>
                    <span className={proc.memPercent > 50 ? "text-green-500" : "text-default"}>
                      {proc.memPercent.toFixed(1)}%
                    </span>
                  </div>
                </td>
                <td className="py-2 text-right text-muted">
                  {formatBytes(proc.memBytes)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {sortedData.length === 0 && (
        <div className="text-center py-8 text-muted">
          No processes match the filter
        </div>
      )}
    </div>
  );
});
