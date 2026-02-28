/**
 * 可观测性 API — 共享类型
 */

export interface ObserveResponse<T> {
  message: string;
  data: T;
}
