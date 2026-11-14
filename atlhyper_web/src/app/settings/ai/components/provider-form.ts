import type { AIProvider, ProviderModelInfo } from "@/api/ai-provider";

/** 新建 Provider 时默认选中的厂商 */
export const DEFAULT_PROVIDER_VENDOR = "gemini";

/** Provider 表单里正在编辑的值 */
export interface ProviderFormDraft {
  name: string;
  provider: string;
  /** 只放用户新输入的 key：已存的 key 不回显，编辑时留空 = 保持不变 */
  apiKey: string;
  /** 从预设列表里选的模型 */
  model: string;
  /** 手填的模型名（预设列表里没有时用） */
  customModel: string;
  useCustomModel: boolean;
  baseUrl: string;
  description: string;
}

/** 某个厂商的预设模型列表 */
export function presetModelsOf(models: ProviderModelInfo[], provider: string): string[] {
  return models.find((entry) => entry.provider === provider)?.models ?? [];
}

/** 打开弹窗时的表单：编辑时按已有的 Provider 填，新建时给一张带默认厂商与模型的空表 */
export function initialProviderForm(editingProvider: AIProvider | null, models: ProviderModelInfo[]): ProviderFormDraft {
  if (editingProvider === null) {
    return {
      name: "",
      provider: DEFAULT_PROVIDER_VENDOR,
      apiKey: "",
      model: presetModelsOf(models, DEFAULT_PROVIDER_VENDOR)[0] ?? "",
      customModel: "",
      useCustomModel: false,
      baseUrl: "",
      description: "",
    };
  }
  // 已存的模型不在预设列表里：当作手填的模型显示
  const isPresetModel = presetModelsOf(models, editingProvider.provider).includes(editingProvider.model);
  return {
    name: editingProvider.name,
    provider: editingProvider.provider,
    apiKey: "",
    model: isPresetModel ? editingProvider.model : "",
    customModel: isPresetModel ? "" : editingProvider.model,
    useCustomModel: !isPresetModel,
    baseUrl: editingProvider.baseUrl || "",
    description: editingProvider.description,
  };
}

/**
 * 换了厂商之后的表单：选中的预设模型不属于新厂商时，换成新厂商的第一个预设模型。
 * 手填模型时不动（手填的名字与厂商的预设列表无关）。
 */
export function withProviderVendor(
  draft: ProviderFormDraft,
  provider: string,
  models: ProviderModelInfo[],
): ProviderFormDraft {
  const presets = presetModelsOf(models, provider);
  const keepsModel = draft.useCustomModel || presets.length === 0 || presets.includes(draft.model);
  return { ...draft, provider, model: keepsModel ? draft.model : presets[0] };
}

/** 切换「手填模型」：关掉手填时回到当前厂商的第一个预设模型 */
export function withCustomModelToggled(
  draft: ProviderFormDraft,
  useCustomModel: boolean,
  models: ProviderModelInfo[],
): ProviderFormDraft {
  const presets = presetModelsOf(models, draft.provider);
  return {
    ...draft,
    useCustomModel,
    model: !useCustomModel && presets.length > 0 ? presets[0] : draft.model,
  };
}
