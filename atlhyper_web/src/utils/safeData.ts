/**
 * 安全数据处理工具
 *
 * 防止后端返回空数据或异常数据导致前端错误
 */

/**
 * 安全获取对象属性，支持嵌套路径
 * @example safeGet(obj, 'a.b.c', defaultValue)
 */
export function safeGet<T>(obj: unknown, path: string, defaultValue: T): T {
  if (obj == null) return defaultValue;

  const keys = path.split('.');
  let result: unknown = obj;

  for (const key of keys) {
    if (result == null || typeof result !== 'object') {
      return defaultValue;
    }
    result = (result as Record<string, unknown>)[key];
  }

  return (result ?? defaultValue) as T;
}

/**
 * 安全数组：确保返回数组类型
 */
export function safeArray<T>(value: unknown, defaultValue: T[] = []): T[] {
  if (Array.isArray(value)) return value;
  return defaultValue;
}

/**
 * 安全数字：确保返回有效数字
 */
export function safeNumber(value: unknown, defaultValue = 0): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string') {
    const num = Number(value);
    if (Number.isFinite(num)) return num;
  }
  return defaultValue;
}

/**
 * 安全字符串：确保返回字符串类型
 */
export function safeString(value: unknown, defaultValue = ''): string {
  if (typeof value === 'string') return value;
  if (value != null) return String(value);
  return defaultValue;
}

/**
 * 安全百分比：确保返回 0-100 范围内的数字
 */
export function safePercent(value: unknown, defaultValue = 0): number {
  const num = safeNumber(value, defaultValue);
  return Math.max(0, Math.min(100, num));
}

/**
 * 安全时间戳转换
 */
export function safeTimestamp(value: unknown): number {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string') {
    const ts = new Date(value).getTime();
    if (Number.isFinite(ts)) return ts;
  }
  return 0;
}

/**
 * 安全数据转换器：统一处理 API 响应
 */
export function safeTransform<T, R>(
  data: T | null | undefined,
  transformer: (data: T) => R,
  defaultValue: R
): R {
  if (data == null) return defaultValue;
  try {
    return transformer(data);
  } catch (error) {
    console.warn('[safeTransform] Transform error:', error);
    return defaultValue;
  }
}
