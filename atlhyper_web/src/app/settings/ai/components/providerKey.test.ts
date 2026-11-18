import { describe, expect, it } from "vitest";
import { apiKeyRequired } from "./providerKey";

describe("apiKeyRequired", () => {
  it("Ollama 自部署、Anthropic（留空走集群身份免密钥）可以不填 key", () => {
    expect(apiKeyRequired("ollama")).toBe(false);
    expect(apiKeyRequired("anthropic")).toBe(false);
  });

  it("其他提供商必须填 key（与后端校验一致）", () => {
    for (const provider of ["openai", "gemini", "bedrock"]) {
      expect(apiKeyRequired(provider)).toBe(true);
    }
  });
});
