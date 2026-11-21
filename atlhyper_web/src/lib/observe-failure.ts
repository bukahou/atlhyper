import type { components } from "@/types/api.gen";
import type { ObserveFailureTranslations } from "@/types/i18n";

/** 观测查询失败的类别，由 api/openapi.yaml 生成（后端 model_v3/command/result_code.go 与之核对） */
export type ObserveFailureCode = components["schemas"]["ObserveFailureCode"];

/** 运行时要逐个比对的取值；satisfies 保证这里只有规格里的取值，下面 switch 的穷尽检查保证一个不漏 */
const OBSERVE_FAILURE_CODES = [
  "not_configured",
  "invalid_params",
  "timeout",
  "unavailable",
  "rejected",
  "failed",
] as const satisfies readonly ObserveFailureCode[];

/** 从请求错误里取出观测失败类别；不是观测接口的失败（响应里没有这几种 code）返回 null */
export function observeFailureCode(error: unknown): ObserveFailureCode | null {
  const code = (error as { response?: { data?: { code?: unknown } } } | null)?.response?.data?.code;
  return typeof code === "string" && (OBSERVE_FAILURE_CODES as readonly string[]).includes(code)
    ? (code as ObserveFailureCode)
    : null;
}

/** 观测失败类别 → 界面提示（⛔ 不显示后端的错误文字）。不是观测失败时返回 null */
export function observeFailureReason(error: unknown, texts: ObserveFailureTranslations): string | null {
  const code = observeFailureCode(error);
  if (code === null) return null;
  switch (code) {
    case "not_configured":
      return texts.notConfigured;
    case "invalid_params":
      return texts.invalidParams;
    case "timeout":
      return texts.timeout;
    case "unavailable":
      return texts.unavailable;
    case "rejected":
      return texts.rejected;
    case "failed":
      return texts.failed;
    default: {
      const unhandled: never = code;
      return unhandled;
    }
  }
}
