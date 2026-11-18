// atlhyper_master_v2/database/errors.go
// 数据库层错误定义
package database

import "errors"

var (
	// ErrIdentityExists 外部身份（issuer + subject）已关联到某个用户
	ErrIdentityExists = errors.New("外部身份已存在")
	// ErrUsernameTaken 用户名已被占用
	ErrUsernameTaken = errors.New("用户名已被占用")
	// ErrClusterExists 集群 ID 已登记
	ErrClusterExists = errors.New("集群已登记")
	// ErrClusterNotFound 集群未登记
	ErrClusterNotFound = errors.New("集群未登记")

	// ErrNotifyReceiverNameTaken 接收人名称已被占用
	ErrNotifyReceiverNameTaken = errors.New("接收人名称已存在")
	// ErrNotifyReceiverNotFound 接收人不存在
	ErrNotifyReceiverNotFound = errors.New("接收人不存在")
	// ErrNotifyRuleNotFound 路由规则不存在
	ErrNotifyRuleNotFound = errors.New("路由规则不存在")
	// ErrNotifyRuleOrderMismatch 重排给出的 id 不是全部规则的一个排列
	ErrNotifyRuleOrderMismatch = errors.New("规则顺序必须恰好包含全部规则各一次")
	// ErrNotifySilenceNotFound 静默不存在
	ErrNotifySilenceNotFound = errors.New("静默不存在")
	// ErrNotifyEncryptionUnavailable 没有配置加密密钥，接收人的凭据无法落库（宁可拒绝，也不存明文）
	ErrNotifyEncryptionUnavailable = errors.New("未配置加密密钥，无法保存接收人配置")
)
