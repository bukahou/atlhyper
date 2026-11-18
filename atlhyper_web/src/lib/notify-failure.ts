/**
 * 通知设置页写操作失败的归类（纯函数）
 */
import { isAxiosError } from "axios";

/** 写操作失败的原因，页面据此选提示语 */
export type NotifyWriteFailure = "conflict" | "notFound" | "invalidInput" | "sendFailed" | "unknown";

/**
 * 把写操作抛出的错误归成几类原因。
 *
 * 按状态码而不是后端的错误文案判断：文案是句子，随时可能改写，状态码是接口约定。
 * 409 = 名称重复或仍被引用；502 = 测试消息发不出去。
 */
export function notifyWriteFailure(error: unknown): NotifyWriteFailure {
  if (!isAxiosError(error)) return "unknown";
  switch (error.response?.status) {
    case 409:
      return "conflict";
    case 404:
      return "notFound";
    case 400:
      return "invalidInput";
    case 502:
      return "sendFailed";
    default:
      return "unknown";
  }
}

/**
 * 后端在失败响应里给出的原因；没有给出时为空串。
 *
 * 删除接收人被拒绝（409）时「被谁引用」、测试发送失败（502）时「为什么失败」只有后端知道，
 * 这两处要把它原样显示出来。错误原因可能在 error 字段，也可能在 message 字段，两个都认。
 */
export function notifyFailureReason(error: unknown): string {
  if (!isAxiosError(error)) return "";
  const body = error.response?.data as { error?: unknown; message?: unknown } | undefined;
  if (typeof body?.error === "string" && body.error !== "") return body.error;
  if (typeof body?.message === "string") return body.message;
  return "";
}

/**
 * 写操作失败时显示给用户的一句话：按失败原因选提示语；后端给了具体原因时附在后面。
 *
 * 409（被谁引用 / 名称重复）、400（哪一项不合规）、502（为什么发不出去）的具体原因只有后端知道。
 */
export function notifyFailureText(error: unknown, messages: Record<NotifyWriteFailure, string>): string {
  const failure = notifyWriteFailure(error);
  const reason = notifyFailureReason(error);
  const reasonIsInformative = failure === "conflict" || failure === "invalidInput" || failure === "sendFailed";
  return reasonIsInformative && reason !== "" ? `${messages[failure]}（${reason}）` : messages[failure];
}
