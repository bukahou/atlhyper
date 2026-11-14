/**
 * 集群注册表的输入校验（纯函数）
 *
 * 规则与后端 model/cluster_registry_validation.go 一致。前端先拦一道只是为了少一次往返、
 * 就地给出提示；后端仍然是唯一的裁决方，所以这里不合规的一定被后端拒绝，反过来不保证。
 */
import { isAxiosError } from "axios";

/** 与 DNS 标签同长：集群 ID 会出现在 URL 路径、请求头与日志里 */
export const CLUSTER_ID_MAX_LENGTH = 63;
/** 显示名与描述的字数上限（按字符计，不按字节） */
export const CLUSTER_NAME_MAX_CHARS = 64;
export const CLUSTER_DESCRIPTION_MAX_CHARS = 256;

/** 首尾是小写字母或数字，中间还可以有 - 和 . */
const CLUSTER_ID_PATTERN = /^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$/;
/** Unicode 控制字符（C0、DEL、C1），对应后端的 unicode.IsControl */
const CONTROL_CHARACTER = /\p{Cc}/u;

export type ClusterIdProblem = "length" | "charset";

/**
 * 集群 ID 哪里不合规；合规返回 null。
 *
 * 不做去空白之类的自动修正：ID 要与 Agent 上报的逐字一致，改写后的 ID 永远对不上。
 */
export function clusterIdProblem(clusterId: string): ClusterIdProblem | null {
  if (clusterId.length === 0 || clusterId.length > CLUSTER_ID_MAX_LENGTH) return "length";
  return CLUSTER_ID_PATTERN.test(clusterId) ? null : "charset";
}

export type ClusterProfileProblem = "nameRequired" | "nameTooLong" | "descriptionTooLong" | "controlCharacter";

/** 按字符数（码点）计长度：一个汉字算一个字，与后端的 utf8.RuneCountInString 一致 */
function characterCount(text: string): number {
  return Array.from(text).length;
}

/**
 * 显示名与描述哪里不合规；合规返回 null。
 *
 * 后端会先去掉两者的首尾空白再校验，这里按去掉之后的内容判断。
 */
export function clusterProfileProblem(name: string, description: string): ClusterProfileProblem | null {
  const trimmedName = name.trim();
  const trimmedDescription = description.trim();
  if (trimmedName === "") return "nameRequired";
  if (characterCount(trimmedName) > CLUSTER_NAME_MAX_CHARS) return "nameTooLong";
  if (characterCount(trimmedDescription) > CLUSTER_DESCRIPTION_MAX_CHARS) return "descriptionTooLong";
  if (CONTROL_CHARACTER.test(trimmedName) || CONTROL_CHARACTER.test(trimmedDescription)) return "controlCharacter";
  return null;
}

/** 写操作失败的原因，页面据此选提示语 */
export type ClusterWriteFailure = "alreadyRegistered" | "notRegistered" | "invalidInput" | "unknown";

/**
 * 把写操作抛出的错误归成几类原因。
 *
 * 按状态码而不是后端的错误文案判断：文案是中文句子，随时可能改写，状态码是接口约定。
 */
export function clusterWriteFailure(error: unknown): ClusterWriteFailure {
  if (!isAxiosError(error)) return "unknown";
  switch (error.response?.status) {
    case 409:
      return "alreadyRegistered";
    case 404:
      return "notRegistered";
    case 400:
      return "invalidInput";
    default:
      return "unknown";
  }
}
