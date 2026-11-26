import { describe, expect, it } from "vitest";
import {
  API_TOKEN_NAME_MAX_CHARS,
  SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS,
  SERVICE_ACCOUNT_NAME_MAX_LENGTH,
  displayNameProblem,
  parseExpiryDays,
  serviceAccountFailure,
  serviceAccountNameProblem,
  tokenRequestProblem,
} from "./service-account-input";

describe("serviceAccountNameProblem", () => {
  it.each(["haro", "abc", "haro-prod-2", "a" + "b".repeat(SERVICE_ACCOUNT_NAME_MAX_LENGTH - 1)])("接受 %s", (name) => {
    expect(serviceAccountNameProblem(name)).toBeNull();
  });

  it("短于 3 个或长于 32 个字符报长度问题", () => {
    expect(serviceAccountNameProblem("")).toBe("nameLength");
    expect(serviceAccountNameProblem("ab")).toBe("nameLength");
    expect(serviceAccountNameProblem("a".repeat(SERVICE_ACCOUNT_NAME_MAX_LENGTH + 1))).toBe("nameLength");
  });

  it.each(["1haro", "-haro", "Haro", "haro_prod", "haro.prod", "haro prod", "语音助手"])("拒绝 %s", (name) => {
    expect(serviceAccountNameProblem(name)).toBe("nameCharset");
  });

  it("不替用户改写：前后带空白直接判不合规", () => {
    expect(serviceAccountNameProblem(" haro")).toBe("nameCharset");
    expect(serviceAccountNameProblem("haro\n")).toBe("nameCharset");
  });
});

describe("displayNameProblem", () => {
  it("可空；按字符计数，首尾空白不计", () => {
    expect(displayNameProblem("")).toBeNull();
    expect(displayNameProblem("语".repeat(SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS))).toBeNull();
    expect(displayNameProblem(`  ${"a".repeat(SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS)}  `)).toBeNull();
    expect(displayNameProblem("语".repeat(SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS + 1))).toBe("displayNameTooLong");
  });

  it("不得含控制字符", () => {
    expect(displayNameProblem("Ha\nro")).toBe("controlCharacter");
    expect(displayNameProblem("Ha\u0000ro")).toBe("controlCharacter");
  });
});

describe("tokenRequestProblem", () => {
  const maxDays = 3650;

  it("名称必填（只有空白也算没填），按字符计数", () => {
    expect(tokenRequestProblem("prod", null, maxDays)).toBeNull();
    expect(tokenRequestProblem("  ", null, maxDays)).toBe("tokenNameRequired");
    expect(tokenRequestProblem("令".repeat(API_TOKEN_NAME_MAX_CHARS), null, maxDays)).toBeNull();
    expect(tokenRequestProblem("令".repeat(API_TOKEN_NAME_MAX_CHARS + 1), null, maxDays)).toBe("tokenNameTooLong");
    expect(tokenRequestProblem("pr\u0007od", null, maxDays)).toBe("controlCharacter");
  });

  it("有效期：不给 = 不过期；给了须为 1 到后端下发的上限", () => {
    expect(tokenRequestProblem("prod", 1, maxDays)).toBeNull();
    expect(tokenRequestProblem("prod", maxDays, maxDays)).toBeNull();
    expect(tokenRequestProblem("prod", 0, maxDays)).toBe("expiryRange");
    expect(tokenRequestProblem("prod", maxDays + 1, maxDays)).toBe("expiryRange");
    expect(tokenRequestProblem("prod", Number.NaN, maxDays)).toBe("expiryRange");
  });

  it("上限跟着后端下发的值走，不写死", () => {
    expect(tokenRequestProblem("prod", 400, 365)).toBe("expiryRange");
    expect(tokenRequestProblem("prod", 365, 365)).toBeNull();
  });
});

describe("parseExpiryDays", () => {
  it("只认整数天；空、小数、带单位的一律 NaN（交给 tokenRequestProblem 报错）", () => {
    expect(parseExpiryDays("90")).toBe(90);
    expect(parseExpiryDays(" 30 ")).toBe(30);
    expect(parseExpiryDays("")).toBeNaN();
    expect(parseExpiryDays("1.5")).toBeNaN();
    expect(parseExpiryDays("30d")).toBeNaN();
    expect(parseExpiryDays("-1")).toBe(-1);
  });
});

describe("serviceAccountFailure", () => {
  const httpError = (status: number) => ({ isAxiosError: true, response: { status } });

  it("409 的含义随操作而定：新建是重名，签发是令牌数已达上限", () => {
    expect(serviceAccountFailure(httpError(409), "create")).toBe("nameTaken");
    expect(serviceAccountFailure(httpError(409), "issue")).toBe("tokenLimitReached");
  });

  it("404 不存在、400 输入不合规", () => {
    for (const operation of ["update", "delete", "issue", "revoke"] as const) {
      expect(serviceAccountFailure(httpError(404), operation)).toBe("notFound");
    }
    expect(serviceAccountFailure(httpError(400), "create")).toBe("invalidInput");
  });

  it("其余状态码与非 HTTP 错误（断网等）一律算未知", () => {
    expect(serviceAccountFailure(httpError(500), "create")).toBe("unknown");
    expect(serviceAccountFailure(httpError(409), "update")).toBe("unknown");
    expect(serviceAccountFailure(new Error("Network Error"), "revoke")).toBe("unknown");
    expect(serviceAccountFailure(undefined, "delete")).toBe("unknown");
  });
});
