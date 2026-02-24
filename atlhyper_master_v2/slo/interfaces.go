// Package slo SLO 领域处理器
//
// interfaces.go - 对外接口定义
//
// slo 包当前包含:
//   - calculator: 纯计算函数（可用性、错误率、状态判断等）
//   - route_updater: 路由映射更新器（从快照同步 IngressRoute 映射）
//
// 时序数据（raw/hourly）已迁移至 OTelSnapshot + ClickHouse，
// 原 processor/aggregator/cleaner/status_checker 已移除。
package slo
