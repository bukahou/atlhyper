import { AxiosError, AxiosHeaders } from "axios";
import { describe, expect, it } from "vitest";
import { createQueryClient, shouldRetry } from "./client";

function httpError(status: number): AxiosError {
  const config = { headers: new AxiosHeaders() };
  return new AxiosError("request failed", "ERR_BAD_REQUEST", config, null, {
    status,
    statusText: "",
    headers: {},
    config,
    data: null,
  });
}

describe("shouldRetry", () => {
  it("网络错误或 5xx 只重试一次", () => {
    expect(shouldRetry(0, new Error("network"))).toBe(true);
    expect(shouldRetry(1, new Error("network"))).toBe(false);
    expect(shouldRetry(0, httpError(503))).toBe(true);
    expect(shouldRetry(1, httpError(503))).toBe(false);
  });

  it("4xx 不重试：鉴权与参数错误重发也不会变", () => {
    for (const status of [400, 401, 403, 404]) {
      expect(shouldRetry(0, httpError(status))).toBe(false);
    }
  });
});

describe("createQueryClient", () => {
  it("标签页不可见时不轮询，切回时刷新", () => {
    const defaults = createQueryClient().getDefaultOptions().queries;
    expect(defaults?.refetchIntervalInBackground).toBe(false);
    expect(defaults?.refetchOnWindowFocus).toBe(true);
  });
});
