"use client";

type StatusType = "success" | "warning" | "error" | "info" | "default";

interface StatusBadgeProps {
  status: string;
  type?: StatusType;
}

const statusColors: Record<StatusType, string> = {
  success: "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400",
  warning: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400",
  error: "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400",
  info: "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400",
  default: "bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300",
};

// 自动推断状态类型
function inferStatusType(status: string): StatusType {
  const lowerStatus = status.toLowerCase();
  if (["running", "ready", "active", "success", "succeeded"].includes(lowerStatus)) {
    return "success";
  }
  if (["pending", "warning", "notready"].includes(lowerStatus)) {
    return "warning";
  }
  if (["failed", "error", "crashloopbackoff"].includes(lowerStatus)) {
    return "error";
  }
  if (["info", "creating", "terminating"].includes(lowerStatus)) {
    return "info";
  }
  return "default";
}

export function StatusBadge({ status, type }: StatusBadgeProps) {
  const statusType = type || inferStatusType(status);

  return (
    <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${statusColors[statusType]}`}>
      {status}
    </span>
  );
}
