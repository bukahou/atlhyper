// AI Chat 类型定义 (前端)

export interface Conversation {
  id: number;
  cluster_id: string;
  title: string;
  message_count: number;
  created_at: string;
  updated_at: string;
}

export interface Message {
  id: number;
  conversation_id: number;
  role: "user" | "assistant";
  content: string;
  tool_calls?: string;
  created_at: string;
}

// 流式渲染段
export interface StreamSegment {
  type: "text" | "tool_call" | "tool_result" | "error";
  content: string;
  tool?: string;
  params?: string;
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
