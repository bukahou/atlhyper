/**
 * AI 对话流（SSE）的读取：分帧交给 eventsource-parser（按规范处理分块、\r\n、多行 data、注释行），
 * 这里只负责把每条事件的 JSON 交给调用方，并判定流是怎么结束的。不依赖 React，便于测试。
 */
import { createParser } from "eventsource-parser";
import type { ChatStats, StreamSegment } from "@/components/ai/types";

/** 流没有正常完成的原因；给人看的文案由调用方按界面语言出 */
export type ChatStreamFailure =
  | { kind: "http"; status: number } // 请求被拒（非 2xx）
  | { kind: "server"; message: string; code?: string } // 后端发来的错误分段（code 为错误码）
  | { kind: "malformed" } // 某条事件不是合法 JSON
  | { kind: "interrupted" } // 没收到 done / error 就断开
  | { kind: "network"; message: string }; // 请求本身没发出去或连接出错

export interface ChatStreamHandlers {
  onSegment: (segment: StreamSegment) => void;
  onDone: (stats?: ChatStats) => void;
}

/**
 * 读完整条流。正常结束（收到 done）返回 null；否则返回原因。
 * 收到 done / error 或遇到格式错误后不再处理后面的内容。
 */
export async function readChatStream(
  body: ReadableStream<Uint8Array>,
  { onSegment, onDone }: ChatStreamHandlers,
): Promise<ChatStreamFailure | null> {
  let outcome: ChatStreamFailure | null | undefined; // undefined = 还没结束

  const parser = createParser({
    onEvent(event) {
      if (outcome !== undefined || event.data === "") return;
      let segment: StreamSegment;
      try {
        segment = JSON.parse(event.data) as StreamSegment;
      } catch {
        outcome = { kind: "malformed" };
        return;
      }
      if (segment.type === "done") {
        outcome = null;
        onDone(segment.stats);
      } else if (segment.type === "error") {
        outcome = { kind: "server", message: segment.content ?? "", code: segment.code };
      } else {
        onSegment(segment);
      }
    },
  });

  const reader = body.getReader();
  const decoder = new TextDecoder();
  try {
    while (outcome === undefined) {
      const { done, value } = await reader.read();
      if (done) {
        parser.feed(decoder.decode()); // 多字节字符被截在最后一块时补齐
        break;
      }
      parser.feed(decoder.decode(value, { stream: true }));
    }
  } finally {
    // 已经有结论时不再等后面的数据
    if (outcome !== undefined) void reader.cancel().catch(() => {});
  }
  return outcome === undefined ? { kind: "interrupted" } : outcome;
}

/** 生成失败提示用到的文案（取自 t.aiChatPage.streamErrors） */
export interface ChatStreamFailureTexts {
  unauthorized: string;
  forbidden: string;
  rateLimited: string;
  httpError: string;
  serverError: string;
  malformed: string;
  interrupted: string;
  network: string;
}

/**
 * 失败原因 → 给人看的提示。
 * 后端错误分段优先按错误码出文案（codeMessage 认得的码）；认不得时用后端给的兜底说明。
 */
export function chatStreamFailureMessage(
  failure: ChatStreamFailure,
  texts: ChatStreamFailureTexts,
  codeMessage: (code: string) => string | undefined = () => undefined,
): string {
  switch (failure.kind) {
    case "http":
      if (failure.status === 401) return texts.unauthorized;
      if (failure.status === 403) return texts.forbidden;
      if (failure.status === 429) return texts.rateLimited;
      return texts.httpError.replace("{status}", String(failure.status));
    case "server":
      return (failure.code && codeMessage(failure.code)) || failure.message || texts.serverError;
    case "malformed":
      return texts.malformed;
    case "interrupted":
      return texts.interrupted;
    case "network":
      return texts.network.replace("{message}", failure.message);
  }
}
