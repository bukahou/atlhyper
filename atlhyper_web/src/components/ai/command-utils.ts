import { StreamSegment } from "./types";

// ==================== 类型定义 ====================

export type CommandStatus = "running" | "success" | "failed";

export interface Command {
  id: string;
  name: string;
  params: string;
  status: CommandStatus;
  result?: string;
}

export interface Round {
  thinking: string;
  commands: Command[];
}

// ==================== 工具函数 ====================

// 检测是否是 ToolResult JSON
export function isToolResultJSON(content: string): boolean {
  if (!content) return false;
  const trimmed = content.trim();
  return trimmed.startsWith('{"CallID":') || trimmed.startsWith('{"callid":');
}

// 格式化指令参数为易读形式
export function formatCommandParams(name: string, paramsJson: string): string {
  try {
    const p = JSON.parse(paramsJson);

    // query_cluster 指令
    if (name === "query_cluster") {
      const action = p.action || "";
      const kind = p.kind || "";
      const ns = p.namespace || "";
      const resourceName = p.name || "";

      // 构建资源路径
      let resource = kind;
      if (ns && resourceName) {
        resource = `${kind} ${ns}/${resourceName}`;
      } else if (resourceName) {
        resource = `${kind} ${resourceName}`;
      } else if (ns) {
        resource = `${kind} -n ${ns}`;
      }

      switch (action) {
        case "describe":
          return `kubectl describe ${resource}`;
        case "get_logs":
          const container = p.container ? ` -c ${p.container}` : "";
          const tail = p.tail ? ` --tail=${p.tail}` : "";
          return `kubectl logs ${resource}${container}${tail}`;
        case "get_events":
          if (ns) {
            return `kubectl get events -n ${ns}${resourceName ? ` --field-selector involvedObject.name=${resourceName}` : ""}`;
          }
          return `kubectl get events`;
        case "list":
          return `kubectl get ${kind}${ns ? ` -n ${ns}` : " -A"}`;
        default:
          return `${action} ${resource}`.trim();
      }
    }

    // 其他指令：简化显示关键字段
    const parts: string[] = [];
    for (const [key, value] of Object.entries(p)) {
      if (value && typeof value === "string" && value.length < 50) {
        parts.push(`${key}=${value}`);
      }
    }
    return parts.join(" ") || paramsJson;
  } catch {
    return paramsJson;
  }
}

// 根据指令生成友好的标题
export function formatCommandTitle(name: string, paramsJson: string): string {
  try {
    const p = JSON.parse(paramsJson);

    if (name === "query_cluster") {
      const action = p.action || "";
      const kind = p.kind || "";
      const resourceName = p.name || "";

      switch (action) {
        case "describe":
          return `查看 ${kind} 详情${resourceName ? `: ${resourceName}` : ""}`;
        case "get_logs":
          return `获取 ${kind} 日志${resourceName ? `: ${resourceName}` : ""}`;
        case "get_events":
          return `查询事件${p.namespace ? ` (${p.namespace})` : ""}`;
        case "list":
          return `列出 ${kind}${p.namespace ? ` (${p.namespace})` : ""}`;
        default:
          return `${action} ${kind}`;
      }
    }

    return name;
  } catch {
    return name;
  }
}

// ==================== 从 StreamSegments 解析 Rounds ====================

export function parseRoundsFromSegments(segments: StreamSegment[]): { rounds: Round[]; finalText: string } {
  const rounds: Round[] = [];
  let currentRound: Round | null = null;
  let finalText = "";
  let pendingToolCall: { name: string; params: string; id: string } | null = null;

  for (const seg of segments) {
    if (seg.type === "text" && !isToolResultJSON(seg.content)) {
      // 文本内容
      if (currentRound) {
        // 如果当前轮有指令，说明这是下一轮的思考
        if (currentRound.commands.length > 0) {
          rounds.push(currentRound);
          currentRound = { thinking: seg.content, commands: [] };
        } else {
          // 追加到当前轮的思考
          currentRound.thinking += seg.content;
        }
      } else {
        // 开始新的一轮
        currentRound = { thinking: seg.content, commands: [] };
      }
      finalText += seg.content;
    } else if (seg.type === "tool_call") {
      // 工具调用开始
      if (!currentRound) {
        currentRound = { thinking: "", commands: [] };
      }
      pendingToolCall = {
        name: seg.tool || "unknown",
        params: seg.params || "{}",
        id: `${seg.tool}-${Date.now()}-${Math.random()}`,
      };
      currentRound.commands.push({
        id: pendingToolCall.id,
        name: pendingToolCall.name,
        params: pendingToolCall.params,
        status: "running",
      });
    } else if (seg.type === "tool_result") {
      // 工具调用结果
      if (currentRound && currentRound.commands.length > 0) {
        // 找到对应的 running 命令并更新状态
        const lastCmd = currentRound.commands.find(
          (c) => c.status === "running" && c.name === seg.tool
        );
        if (lastCmd) {
          lastCmd.status = "success";
          lastCmd.result = seg.content;
        }
      }
      pendingToolCall = null;
    }
  }

  // 保存最后一轮
  if (currentRound && (currentRound.thinking || currentRound.commands.length > 0)) {
    rounds.push(currentRound);
  }

  return { rounds, finalText };
}
