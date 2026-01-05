/**
 * ============================================================
 * [TEST] 权限测试 API
 * ============================================================
 *
 * 这是测试代码，用于验证前端权限处理机制
 * 功能开发完成后需要删除
 *
 * 权限分类：
 * 1. 公开接口 - 无需登录，网站整体使用（查看数据）
 * 2. 操作接口 - 需要登录，资源操作（restart/scale/cordon 等）
 * 3. 管理接口 - 需要 Admin 权限
 *
 * 预期结果：
 * - 公开接口：始终成功
 * - 操作接口：未登录返回 401，登录后成功
 * - 管理接口：未登录返回 401，非 Admin 返回 403
 */

import { post } from "./request";
import { getCurrentClusterId } from "@/config/cluster";

/**
 * [TEST] 测试公开接口（无需登录）
 * 查看类接口，任何人都可以访问
 */
export function testPublicApi() {
  return post("/uiapi/cluster/overview", { ClusterID: getCurrentClusterId() });
}

/**
 * [TEST] 测试操作接口（需要登录）
 * 未登录时返回 401
 */
export function testOperatorApi() {
  return post("/uiapi/ops/pod/restart", {
    clusterID: getCurrentClusterId(),
    namespace: "default",
    pod: "test-pod-nonexistent",
  });
}

/**
 * [TEST] 测试管理接口（需要 Admin 权限）
 * 未登录返回 401，非 Admin 返回 403
 */
export function testAdminApi() {
  return post("/uiapi/auth/user/update-role", {
    userId: 999,
    role: 1,
  });
}
