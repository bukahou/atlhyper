import type { Message, StreamNotice, StreamSegment } from "./types";

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

/** 一次回答：各轮（有工具调用的那几次调模型）+ 最终回答（最后一次调模型的正文，还没调用工具） */
export interface ParsedAnswer {
  rounds: Round[];
  finalText: string;
  notices: StreamNotice[];
}

/**
 * 流式分段 → 各轮。轮次按分段自带的 round 分、调用与结果按 callId 配对、失败看 isError ——
 * ⛔ 不按文字出现的顺序猜轮次、不按工具名配对（同一轮可能并行调用同名工具）。
 */
export function parseRoundsFromSegments(segments: StreamSegment[]): ParsedAnswer {
  const byRound = new Map<number, Round>();
  const order: number[] = [];
  const notices: StreamNotice[] = [];
  const roundOf = (n: number): Round => {
    let r = byRound.get(n);
    if (!r) {
      r = { thinking: "", commands: [] };
      byRound.set(n, r);
      order.push(n);
    }
    return r;
  };

  for (const seg of segments) {
    const n = seg.round ?? 0;
    switch (seg.type) {
      case "text":
        roundOf(n).thinking += seg.content ?? "";
        break;
      case "tool_call": {
        const round = roundOf(n);
        round.commands.push({
          id: seg.callId || `${n}-${round.commands.length}`,
          name: seg.tool || "unknown",
          params: seg.params || "{}",
          status: "running",
        });
        break;
      }
      case "tool_result": {
        const cmd = roundOf(n).commands.find((c) => c.id === seg.callId);
        if (cmd) {
          cmd.status = seg.isError ? "failed" : "success";
          cmd.result = seg.content;
        }
        break;
      }
      case "notice":
        if (seg.notice) notices.push(seg.notice);
        break;
    }
  }

  const all = order.map((n) => byRound.get(n)!);
  return splitFinal(all, notices);
}

/** 最后一轮没有工具调用时，它的正文就是最终回答；其余有调用的轮进执行块 */
function splitFinal(all: Round[], notices: StreamNotice[]): ParsedAnswer {
  const last = all[all.length - 1];
  if (last && last.commands.length === 0) {
    return { rounds: all.slice(0, -1).filter((r) => r.commands.length > 0), finalText: last.thinking, notices };
  }
  return { rounds: all.filter((r) => r.commands.length > 0), finalText: "", notices };
}

// ==================== 从历史消息解析 ====================

/** 对话里显示的一项：用户消息，或一次完整回答 */
export type ChatItem =
  | { kind: "user"; key: string; message: Message }
  | { kind: "answer"; key: string; answer: ParsedAnswer };

/**
 * 历史消息 → 显示项。一次回答 = 一条用户消息之后的全部 assistant / tool 消息：
 * 带工具调用的 assistant 消息各是一轮，tool 消息按 callId 对回各自的调用，
 * 最后一条不带调用的 assistant 消息是最终回答。
 */
export function groupConversation(messages: Message[]): ChatItem[] {
  const items: ChatItem[] = [];
  let rounds: Round[] = [];
  let finalText = "";
  let answerKey = "";

  const flush = () => {
    // 历史里没有结果的调用：那次回答中途断了，按失败显示（⛔ 不显示成一直在执行）
    for (const round of rounds) {
      for (const cmd of round.commands) {
        if (cmd.status === "running") cmd.status = "failed";
      }
    }
    if (rounds.length > 0 || finalText) {
      items.push({ kind: "answer", key: answerKey, answer: { rounds, finalText, notices: [] } });
    }
    rounds = [];
    finalText = "";
  };

  for (const msg of messages) {
    if (msg.role === "user") {
      flush();
      items.push({ kind: "user", key: `u-${msg.id}`, message: msg });
      continue;
    }
    if (!answerKey || (rounds.length === 0 && !finalText)) answerKey = `a-${msg.id}`;
    if (msg.role === "assistant") {
      if (msg.toolCalls && msg.toolCalls.length > 0) {
        rounds.push({
          thinking: msg.content,
          commands: msg.toolCalls.map((tc) => ({ id: tc.id, name: tc.name, params: tc.params, status: "running" })),
        });
      } else if (msg.content) {
        finalText = finalText ? `${finalText}\n\n${msg.content}` : msg.content;
      }
    } else if (msg.role === "tool" && msg.toolResult) {
      const result = msg.toolResult;
      for (const round of rounds) {
        const cmd = round.commands.find((c) => c.id === result.callId);
        if (cmd) {
          cmd.status = result.isError ? "failed" : "success";
          cmd.result = result.content;
        }
      }
    }
  }
  flush();
  return items;
}
