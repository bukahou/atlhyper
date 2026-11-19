// 集群健康结论 → 颜色等级（判定在后端，这里只做取值换算）
import type { HardwareStatus } from "@/types/hardware";
import type { ClusterHealthStatus } from "@/types/overview";

export function clusterHealthLevel(status: ClusterHealthStatus): HardwareStatus {
  switch (status) {
    case "Healthy":
      return "good";
    case "Degraded":
      return "warn";
    case "Unhealthy":
      return "crit";
    default:
      return "unknown";
  }
}
