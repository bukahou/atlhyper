// AI Chat 类型定义 (前端)

export interface Conversation {
  id: number;
  clusterId: string;
  title: string;
  messageCount: number;
  // 累计统计
  totalInputTokens: number;  // 累计输入 Token
  totalOutputTokens: number; // 累计输出 Token
  totalToolCalls: number;    // 累计指令数
  createdAt: string;
  updatedAt: string;
}

/** 模型发起的一次工具调用（对齐后端 llm.ToolCall） */
export interface ToolCallRecord {
  id: string;
  name: string;
  params: string;
}

/** 一次工具调用的结果（对齐后端 llm.ToolResult），按 callId 对应到调用 */
export interface ToolResultRecord {
  callId: string;
  name: string;
  content: string;
  isError?: boolean;
}

export interface Message {
  id: number;
  conversationId: number;
  role: "user" | "assistant" | "tool";
  /** user / assistant 的正文；tool 消息为空（结果在 toolResult） */
  content: string;
  /** 带工具调用的 assistant 消息 = 一轮 */
  toolCalls?: ToolCallRecord[];
  toolResult?: ToolResultRecord;
  createdAt: string;
}

// 流式渲染段
/** 流里的提示代码（文案在 i18n aiChatPage.notice） */
export type StreamNotice = "historyTrimmed" | "toolCallsCapped";

// 流式渲染段（对齐后端 ai.ChatChunk）
export interface StreamSegment {
  type: "text" | "tool_call" | "tool_result" | "notice" | "done" | "error";
  /** 第几次调模型（从 1 开始）：按它分轮 */
  round?: number;
  content?: string;
  /** tool_call 与 tool_result 按它配对（同一轮可能并行调用同名工具） */
  callId?: string;
  tool?: string;
  params?: string;
  /** tool_result：执行失败 */
  isError?: boolean;
  notice?: StreamNotice;
  stats?: ChatStats; // done 时返回的统计信息
}

// 解析后的工具调用
export interface ToolCall {
  id: string;
  name: string;
  params: string;
  status: "running" | "success" | "failed";
  result?: string;
}

// 思考轮次（一次 AI 输出 + 相关工具调用）
export interface ThinkingRound {
  thinking: string;       // AI 的思考/回复文本
  toolCalls: ToolCall[];  // 该轮调用的工具
}

// 单次提问的统计信息（后端 done 时返回）
export interface ChatStats {
  rounds: number;          // 思考轮次（AI 调用次数）
  totalToolCalls: number;  // 总指令数（所有轮次的 Tool 调用总数）
  inputTokens: number;     // 输入 Token 数
  outputTokens: number;    // 输出 Token 数
}

// 单次 API 调用的 Token 用量
export interface CallTokenUsage {
  callIndex: number;       // 第几次调用 (从 1 开始)
  inputTokens: number;     // 输入 token (system + tools + context)
  outputTokens: number;    // 输出 token (response)
  totalTokens: number;     // 本次调用 input + output
}

// 对话级别的 Token 统计
export interface ConversationTokens {
  totalInput: number;      // 累计输入 token
  totalOutput: number;     // 累计输出 token
  total: number;           // 累计总 token
  calls: CallTokenUsage[]; // 每次调用详情
}
