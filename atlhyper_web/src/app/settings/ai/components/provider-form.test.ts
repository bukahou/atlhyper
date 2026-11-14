import { describe, expect, it } from "vitest";
import type { AIProvider, ProviderModelInfo } from "@/api/ai-provider";
import { initialProviderForm, presetModelsOf, withCustomModelToggled, withProviderVendor } from "./provider-form";

const models = [
  { provider: "gemini", models: ["gemini-a", "gemini-b"] },
  { provider: "openai", models: ["gpt-a"] },
  { provider: "ollama", models: [] },
] as unknown as ProviderModelInfo[];

function savedProvider(overrides: Partial<AIProvider>): AIProvider {
  return {
    id: 1,
    name: "main",
    provider: "openai",
    model: "gpt-a",
    baseUrl: "",
    description: "desc",
    ...overrides,
  } as AIProvider;
}

describe("initialProviderForm", () => {
  it("新建：默认厂商 + 它的第一个预设模型，其余为空", () => {
    expect(initialProviderForm(null, models)).toEqual({
      name: "",
      provider: "gemini",
      apiKey: "",
      model: "gemini-a",
      customModel: "",
      useCustomModel: false,
      baseUrl: "",
      description: "",
    });
  });

  it("编辑：按已有的 Provider 填，key 不回显", () => {
    const draft = initialProviderForm(savedProvider({}), models);
    expect(draft).toMatchObject({ name: "main", provider: "openai", model: "gpt-a", useCustomModel: false, apiKey: "" });
  });

  it("编辑：已存的模型不在预设列表里时当作手填", () => {
    const draft = initialProviderForm(savedProvider({ model: "my-finetune" }), models);
    expect(draft).toMatchObject({ model: "", customModel: "my-finetune", useCustomModel: true });
  });
});

describe("withProviderVendor", () => {
  const draft = initialProviderForm(null, models);

  it("换厂商后预设模型不属于新厂商 → 换成新厂商的第一个", () => {
    expect(withProviderVendor(draft, "openai", models)).toMatchObject({ provider: "openai", model: "gpt-a" });
  });

  it("手填模型、或新厂商没有预设列表时不动模型", () => {
    const custom = { ...draft, useCustomModel: true, customModel: "x" };
    expect(withProviderVendor(custom, "openai", models)).toMatchObject({ provider: "openai", model: "gemini-a" });
    expect(withProviderVendor(draft, "ollama", models)).toMatchObject({ provider: "ollama", model: "gemini-a" });
  });
});

describe("withCustomModelToggled / presetModelsOf", () => {
  it("关掉手填回到当前厂商的第一个预设模型；打开手填不动预设模型", () => {
    const custom = { ...initialProviderForm(null, models), useCustomModel: true, model: "" };
    expect(withCustomModelToggled(custom, false, models)).toMatchObject({ useCustomModel: false, model: "gemini-a" });
    expect(withCustomModelToggled(initialProviderForm(null, models), true, models)).toMatchObject({
      useCustomModel: true,
      model: "gemini-a",
    });
  });

  it("没有登记的厂商返回空列表", () => {
    expect(presetModelsOf(models, "unknown")).toEqual([]);
  });
});
