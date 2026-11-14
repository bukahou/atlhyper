import { AxiosError, CanceledError } from "axios";
import { describe, expect, it } from "vitest";
import { shouldReportRequestFailure } from "./request";

describe("shouldReportRequestFailure", () => {
  it("被取消的请求不上报：取消是数据层的正常行为，不是错误", () => {
    expect(shouldReportRequestFailure(new CanceledError("canceled"))).toBe(false);
    expect(shouldReportRequestFailure(new AxiosError("canceled", "ERR_CANCELED"))).toBe(false);
    // 中止信号在请求发出前就触发时，拿到的可能只是带 code 的普通对象
    expect(shouldReportRequestFailure({ code: "ERR_CANCELED" })).toBe(false);
  });

  it("其他失败照常上报", () => {
    expect(shouldReportRequestFailure(new AxiosError("Network Error", "ERR_NETWORK"))).toBe(true);
    expect(shouldReportRequestFailure(new AxiosError("timeout", "ECONNABORTED"))).toBe(true);
    expect(shouldReportRequestFailure(new AxiosError("Request failed with status code 500", "ERR_BAD_RESPONSE"))).toBe(true);
    expect(shouldReportRequestFailure(new Error("boom"))).toBe(true);
    expect(shouldReportRequestFailure(undefined)).toBe(true);
  });
});
