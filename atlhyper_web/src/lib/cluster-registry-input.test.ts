import { describe, expect, it } from "vitest";
import {
  CLUSTER_DESCRIPTION_MAX_CHARS,
  CLUSTER_ID_MAX_LENGTH,
  CLUSTER_NAME_MAX_CHARS,
  clusterIdProblem,
  clusterProfileProblem,
  clusterWriteFailure,
} from "./cluster-registry-input";

describe("clusterIdProblem", () => {
  it.each(["c1", "a", "0", "prod-tokyo", "edge.site-01", "a.b", "a".repeat(CLUSTER_ID_MAX_LENGTH)])(
    "接受 %s",
    (clusterId) => {
      expect(clusterIdProblem(clusterId)).toBeNull();
    },
  );

  it("空串与超过 63 个字符的 ID 报长度问题", () => {
    expect(clusterIdProblem("")).toBe("length");
    expect(clusterIdProblem("a".repeat(CLUSTER_ID_MAX_LENGTH + 1))).toBe("length");
  });

  it.each(["Prod", "c_1", "c 1", " c1", "c1 ", "集群", "c1/x", "c%31"])("拒绝含不允许字符的 %s", (clusterId) => {
    expect(clusterIdProblem(clusterId)).toBe("charset");
  });

  it.each(["-c1", "c1-", ".c1", "c1.", "-", "."])("首尾必须是字母或数字：拒绝 %s", (clusterId) => {
    expect(clusterIdProblem(clusterId)).toBe("charset");
  });

  it("不替用户改写：前后带空白的 ID 直接判不合规，而不是去掉空白后放行", () => {
    // ID 要与 Agent 上报的逐字一致，悄悄改写后的 ID 永远对不上
    expect(clusterIdProblem("c1\n")).toBe("charset");
  });
});

describe("clusterProfileProblem", () => {
  it("显示名必填；只有空白也算没填（后端会去掉首尾空白）", () => {
    expect(clusterProfileProblem("", "")).toBe("nameRequired");
    expect(clusterProfileProblem("   ", "说明")).toBe("nameRequired");
    expect(clusterProfileProblem("生产", "")).toBeNull();
  });

  it("字数按字符计，不按字节：64 个汉字的显示名合规，65 个超限", () => {
    expect(clusterProfileProblem("生".repeat(CLUSTER_NAME_MAX_CHARS), "")).toBeNull();
    expect(clusterProfileProblem("生".repeat(CLUSTER_NAME_MAX_CHARS + 1), "")).toBe("nameTooLong");
  });

  it("首尾空白不计入字数", () => {
    expect(clusterProfileProblem(`  ${"a".repeat(CLUSTER_NAME_MAX_CHARS)}  `, "")).toBeNull();
  });

  it("描述可空，超过 256 个字报超限", () => {
    expect(clusterProfileProblem("生产", "述".repeat(CLUSTER_DESCRIPTION_MAX_CHARS))).toBeNull();
    expect(clusterProfileProblem("生产", "述".repeat(CLUSTER_DESCRIPTION_MAX_CHARS + 1))).toBe("descriptionTooLong");
  });

  it("显示名与描述都不得含控制字符", () => {
    expect(clusterProfileProblem("生\u0000产", "")).toBe("controlCharacter");
    expect(clusterProfileProblem("生产", "第一行\n第二行")).toBe("controlCharacter");
    expect(clusterProfileProblem("生产", "a\u007fb")).toBe("controlCharacter");
  });
});

describe("clusterWriteFailure", () => {
  const httpError = (status: number) => ({ isAxiosError: true, response: { status } });

  it("按状态码区分：409 已登记、404 未登记、400 输入不合规", () => {
    expect(clusterWriteFailure(httpError(409))).toBe("alreadyRegistered");
    expect(clusterWriteFailure(httpError(404))).toBe("notRegistered");
    expect(clusterWriteFailure(httpError(400))).toBe("invalidInput");
  });

  it("其余状态码与非 HTTP 错误（断网等）一律算未知", () => {
    expect(clusterWriteFailure(httpError(500))).toBe("unknown");
    expect(clusterWriteFailure(new Error("Network Error"))).toBe("unknown");
    expect(clusterWriteFailure(undefined)).toBe("unknown");
  });
});
