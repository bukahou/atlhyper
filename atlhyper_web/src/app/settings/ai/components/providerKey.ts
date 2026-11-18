/**
 * 新建提供商时可以不填 key 的类型：Ollama 自部署；Anthropic 留空则用集群身份免密钥访问
 * （Workload Identity Federation，由后端的 SDK 凭证链处理）。与后端 apiKeyRequired 一致。
 */
const KEYLESS_PROVIDERS: ReadonlySet<string> = new Set(["ollama", "anthropic"]);

export function apiKeyRequired(provider: string): boolean {
  return !KEYLESS_PROVIDERS.has(provider);
}
