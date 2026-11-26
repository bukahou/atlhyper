/**
 * 服务账号与访问令牌的输入校验（纯函数）
 *
 * 格式规则与后端 model/service_account_token_validation.go 一致。前端先拦一道只是为了少一次往返、
 * 就地给出提示；后端仍然是唯一的裁决方，所以这里不合规的一定被后端拒绝，反过来不保证。
 * 业务参数（可选等级、令牌有效期上限、令牌数上限）由后端随列表下发，这里不写死。
 */
import { isAxiosError } from "axios";

export const SERVICE_ACCOUNT_NAME_MIN_LENGTH = 3;
export const SERVICE_ACCOUNT_NAME_MAX_LENGTH = 32;
/** 显示名与令牌名称的字数上限（按字符计，不按字节） */
export const SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS = 64;
export const API_TOKEN_NAME_MAX_CHARS = 64;

/** 小写字母开头，其后是小写字母、数字或连字符 */
const SERVICE_ACCOUNT_NAME_PATTERN = /^[a-z][a-z0-9-]*$/;
/** Unicode 控制字符（C0、DEL、C1），对应后端的 unicode.IsControl */
const CONTROL_CHARACTER = /\p{Cc}/u;

/** 按字符数（码点）计长度：一个汉字算一个字，与后端的 utf8.RuneCountInString 一致 */
function characterCount(text: string): number {
  return Array.from(text).length;
}

export type ServiceAccountNameProblem = "nameLength" | "nameCharset";

/**
 * 账号名哪里不合规；合规返回 null。
 *
 * 不做去空白之类的自动修正：账号名会出现在审计与日志里，改写过的名字与输入的对不上。
 */
export function serviceAccountNameProblem(name: string): ServiceAccountNameProblem | null {
  if (name.length < SERVICE_ACCOUNT_NAME_MIN_LENGTH || name.length > SERVICE_ACCOUNT_NAME_MAX_LENGTH) return "nameLength";
  return SERVICE_ACCOUNT_NAME_PATTERN.test(name) ? null : "nameCharset";
}

export type DisplayNameProblem = "displayNameTooLong" | "controlCharacter";

/** 显示名哪里不合规（可空；后端会先去掉首尾空白，这里按去掉之后的内容判断）；合规返回 null */
export function displayNameProblem(displayName: string): DisplayNameProblem | null {
  const trimmed = displayName.trim();
  if (characterCount(trimmed) > SERVICE_ACCOUNT_DISPLAY_NAME_MAX_CHARS) return "displayNameTooLong";
  return CONTROL_CHARACTER.test(trimmed) ? "controlCharacter" : null;
}

export type TokenRequestProblem = "tokenNameRequired" | "tokenNameTooLong" | "controlCharacter" | "expiryRange";

/**
 * 签发令牌的输入哪里不合规；合规返回 null。
 *
 * expiresInDays 为 null 表示不过期；maxExpiresInDays 取后端下发的业务参数。
 */
export function tokenRequestProblem(
  name: string,
  expiresInDays: number | null,
  maxExpiresInDays: number,
): TokenRequestProblem | null {
  const trimmed = name.trim();
  if (trimmed === "") return "tokenNameRequired";
  if (characterCount(trimmed) > API_TOKEN_NAME_MAX_CHARS) return "tokenNameTooLong";
  if (CONTROL_CHARACTER.test(trimmed)) return "controlCharacter";
  if (expiresInDays !== null && !(Number.isInteger(expiresInDays) && expiresInDays >= 1 && expiresInDays <= maxExpiresInDays)) {
    return "expiryRange";
  }
  return null;
}

/** 有效期输入框的内容 → 天数。只认整数；空串、小数、带单位的返回 NaN（由 tokenRequestProblem 报出） */
export function parseExpiryDays(text: string): number {
  const trimmed = text.trim();
  return /^-?\d+$/.test(trimmed) ? Number(trimmed) : Number.NaN;
}

export type ServiceAccountOperation = "create" | "update" | "delete" | "issue" | "revoke";
export type ServiceAccountFailure = "nameTaken" | "tokenLimitReached" | "notFound" | "invalidInput" | "unknown";

/**
 * 把写操作抛出的错误归成几类原因，页面据此选提示语。
 *
 * 按状态码而不是后端的错误文案判断：文案是中文句子，随时可能改写，状态码是接口约定。
 * 409 只有两种：新建时账号名被占、签发时未吊销的令牌已达上限。
 */
export function serviceAccountFailure(error: unknown, operation: ServiceAccountOperation): ServiceAccountFailure {
  if (!isAxiosError(error)) return "unknown";
  switch (error.response?.status) {
    case 409:
      if (operation === "create") return "nameTaken";
      if (operation === "issue") return "tokenLimitReached";
      return "unknown";
    case 404:
      return "notFound";
    case 400:
      return "invalidInput";
    default:
      return "unknown";
  }
}
