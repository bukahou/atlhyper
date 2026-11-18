import { AxiosError, type AxiosResponse } from "axios";
import { describe, expect, it } from "vitest";
import { notifyFailureReason, notifyFailureText, notifyWriteFailure } from "./notify-failure";

function httpError(status: number, body: unknown = {}): AxiosError {
  const response = { status, data: body } as AxiosResponse;
  return new AxiosError("request failed", "ERR_BAD_RESPONSE", undefined, undefined, response);
}

describe("notifyWriteFailure", () => {
  it.each([
    [409, "conflict"],
    [404, "notFound"],
    [400, "invalidInput"],
    [502, "sendFailed"],
    [500, "unknown"],
  ])("状态码 %i 归为 %s", (status, failure) => {
    expect(notifyWriteFailure(httpError(status))).toBe(failure);
  });

  it("不是 HTTP 错误（网络断开、代码抛错）一律归为 unknown", () => {
    expect(notifyWriteFailure(new Error("boom"))).toBe("unknown");
    expect(notifyWriteFailure(undefined)).toBe("unknown");
  });
});

describe("notifyFailureReason", () => {
  it("取后端错误响应里的 error 字段", () => {
    expect(notifyFailureReason(httpError(409, { error: "仍被规则「生产高严重度」引用" }))).toBe(
      "仍被规则「生产高严重度」引用",
    );
  });

  it("没有 error 时取 message", () => {
    expect(notifyFailureReason(httpError(502, { message: "smtp: connection refused" }))).toBe(
      "smtp: connection refused",
    );
  });

  it("两个都有时以 error 为准", () => {
    expect(notifyFailureReason(httpError(502, { error: "a", message: "b" }))).toBe("a");
  });

  it("后端没给原因、响应体不是对象、根本不是 HTTP 错误：空串", () => {
    expect(notifyFailureReason(httpError(500, {}))).toBe("");
    expect(notifyFailureReason(httpError(500, "gateway timeout"))).toBe("");
    expect(notifyFailureReason(new Error("boom"))).toBe("");
  });
});

describe("notifyFailureText", () => {
  const messages = {
    conflict: "冲突",
    notFound: "不存在",
    invalidInput: "输入不合规",
    sendFailed: "发送失败",
    unknown: "操作失败",
  };

  it("409 / 400 / 502：后端给了原因就附在提示语后面", () => {
    expect(notifyFailureText(httpError(409, { error: "仍被规则引用" }), messages)).toBe("冲突（仍被规则引用）");
    expect(notifyFailureText(httpError(400, { error: "端口无效" }), messages)).toBe("输入不合规（端口无效）");
    expect(notifyFailureText(httpError(502, { error: "连接超时" }), messages)).toBe("发送失败（连接超时）");
  });

  it("后端没给原因时只有提示语", () => {
    expect(notifyFailureText(httpError(409), messages)).toBe("冲突");
  });

  it("404 与其他错误不附后端文案：那不是给用户看的原因", () => {
    expect(notifyFailureText(httpError(404, { error: "receiver not found" }), messages)).toBe("不存在");
    expect(notifyFailureText(httpError(500, { error: "internal error" }), messages)).toBe("操作失败");
    expect(notifyFailureText(new Error("boom"), messages)).toBe("操作失败");
  });
});
