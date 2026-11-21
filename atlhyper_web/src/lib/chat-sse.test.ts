import { describe, expect, it, vi } from "vitest";
import { chatStreamFailureMessage, readChatStream } from "./chat-sse";

/** 把若干段字节拼成一条可读流（模拟网络分块） */
function streamOf(...chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder();
  return new ReadableStream({
    start(controller) {
      for (const chunk of chunks) controller.enqueue(encoder.encode(chunk));
      controller.close();
    },
  });
}

function handlers() {
  return { onSegment: vi.fn(), onDone: vi.fn() };
}

describe("readChatStream", () => {
  it("一条事件被拆在两个分块里也能完整拿到", async () => {
    const h = handlers();
    const failure = await readChatStream(
      streamOf('data: {"type":"text","round":1,"con', 'tent":"你好"}\n\n', 'data: {"type":"done","stats":{"rounds":1,"totalToolCalls":0,"inputTokens":1,"outputTokens":2}}\n\n'),
      h,
    );
    expect(failure).toBeNull();
    expect(h.onSegment).toHaveBeenCalledWith({ type: "text", round: 1, content: "你好" });
    expect(h.onDone).toHaveBeenCalledWith({ rounds: 1, totalToolCalls: 0, inputTokens: 1, outputTokens: 2 });
  });

  it("\\r\\n 换行、多行 data、注释行按规范处理", async () => {
    const h = handlers();
    const failure = await readChatStream(
      streamOf(': keep-alive\r\n\r\n', 'data: {"type":"text",\r\ndata: "content":"a"}\r\n\r\n', 'data: {"type":"done"}\r\n\r\n'),
      h,
    );
    expect(failure).toBeNull();
    expect(h.onSegment).toHaveBeenCalledTimes(1);
    expect(h.onSegment).toHaveBeenCalledWith({ type: "text", content: "a" });
    expect(h.onDone).toHaveBeenCalledTimes(1);
  });

  it("后端的错误分段：结束并带回错误码与兜底说明，不再调 onDone", async () => {
    const h = handlers();
    const failure = await readChatStream(streamOf('data: {"type":"error","code":"ai_rate_limited","content":"请求频率或配额超限"}\n\n'), h);
    expect(failure).toEqual({ kind: "server", message: "请求频率或配额超限", code: "ai_rate_limited" });
    expect(h.onDone).not.toHaveBeenCalled();
  });

  it("事件不是合法 JSON：按格式错误结束，⛔ 不静默跳过", async () => {
    const h = handlers();
    const failure = await readChatStream(streamOf('data: {"type":"text"\n\n', 'data: {"type":"done"}\n\n'), h);
    expect(failure).toEqual({ kind: "malformed" });
    expect(h.onDone).not.toHaveBeenCalled();
  });

  it("没收到 done / error 就断开：是中断，不是正常结束", async () => {
    const h = handlers();
    const failure = await readChatStream(streamOf('data: {"type":"text","content":"半句"}\n\n'), h);
    expect(failure).toEqual({ kind: "interrupted" });
    expect(h.onSegment).toHaveBeenCalledTimes(1);
  });

  it("done 之后的内容不再处理", async () => {
    const h = handlers();
    await readChatStream(streamOf('data: {"type":"done"}\n\ndata: {"type":"text","content":"x"}\n\n'), h);
    expect(h.onSegment).not.toHaveBeenCalled();
  });
});

describe("chatStreamFailureMessage", () => {
  const texts = {
    unauthorized: "401", forbidden: "403", rateLimited: "429", httpError: "http {status}",
    serverError: "server", malformed: "malformed", interrupted: "interrupted", network: "net {message}",
  };
  it.each([
    [{ kind: "http", status: 401 } as const, "401"],
    [{ kind: "http", status: 403 } as const, "403"],
    [{ kind: "http", status: 429 } as const, "429"],
    [{ kind: "http", status: 502 } as const, "http 502"],
    [{ kind: "server", message: "模型不可用" } as const, "模型不可用"],
    [{ kind: "server", message: "" } as const, "server"],
    [{ kind: "malformed" } as const, "malformed"],
    [{ kind: "interrupted" } as const, "interrupted"],
    [{ kind: "network", message: "Failed to fetch" } as const, "net Failed to fetch"],
  ])("%o → %s", (failure, want) => {
    expect(chatStreamFailureMessage(failure, texts)).toBe(want);
  });
});

describe("readChatStream 字节边界", () => {
  it("一个中文字符的 UTF-8 字节被拆在两个分块里", async () => {
    const bytes = new TextEncoder().encode('data: {"type":"text","content":"你好"}\n\ndata: {"type":"done"}\n\n');
    const cut = bytes.indexOf(0xe4) + 1; // 「你」的第一个字节之后
    const body = new ReadableStream<Uint8Array>({
      start(controller) {
        controller.enqueue(bytes.slice(0, cut));
        controller.enqueue(bytes.slice(cut));
        controller.close();
      },
    });
    const onSegment = vi.fn();
    expect(await readChatStream(body, { onSegment, onDone: vi.fn() })).toBeNull();
    expect(onSegment).toHaveBeenCalledWith({ type: "text", content: "你好" });
  });
});

describe("chatStreamFailureMessage 错误码", () => {
  const texts = {
    unauthorized: "401", forbidden: "403", rateLimited: "429", httpError: "http {status}",
    serverError: "server", malformed: "malformed", interrupted: "interrupted", network: "net {message}",
  };
  const known = (code: string) => (code === "ai_auth" ? "检查密钥" : undefined);
  it("认得的码按码出文案", () => {
    expect(chatStreamFailureMessage({ kind: "server", message: "兜底", code: "ai_auth" }, texts, known)).toBe("检查密钥");
  });
  it("认不得的码用后端的兜底说明", () => {
    expect(chatStreamFailureMessage({ kind: "server", message: "兜底", code: "ai_new_code" }, texts, known)).toBe("兜底");
  });
});
