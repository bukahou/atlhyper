/**
 * AI 调用失败 → 可操作的提示文案
 *
 * 后端以稳定错误码返回失败原因（atlhyper_master_v2/gateway/handler/aiops/ai_error.go），
 * 前端只做「码 → i18n 文案」映射；未知码退回通用文案。
 */

export interface AIErrorTexts {
  analysisFailed: string;
  errorRoleUnassigned: string;
  errorBudgetExhausted: string;
  errorUpstream: string;
  errorTimeout: string;
  analysisInterrupted: string;
}

/** HTTP 失败（axios 错误）→ 文案 */
export function aiErrorMessage(err: unknown, texts: AIErrorTexts): string {
  const code = (err as { response?: { data?: { code?: string } } })?.response?.data?.code;
  return aiCodeMessage(code, texts);
}

/** 错误码 → 文案（HTTP 响应的 code 与深度分析状态的 errorCode 同一套） */
export function aiCodeMessage(code: string | undefined, texts: AIErrorTexts): string {
  switch (code) {
    case "ai_role_unassigned":
      return texts.errorRoleUnassigned;
    case "ai_budget_exhausted":
      return texts.errorBudgetExhausted;
    case "ai_upstream":
      return texts.errorUpstream;
    case "ai_timeout":
      return texts.errorTimeout;
    case "analysis_interrupted":
      return texts.analysisInterrupted;
    default:
      return texts.analysisFailed;
  }
}
