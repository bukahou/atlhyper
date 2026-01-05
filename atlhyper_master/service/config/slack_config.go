// atlhyper_master/service/config/slack_config.go
package config

import (
	dbcfg "AtlHyper/atlhyper_master/db/repository/config"
)

// UI 层读取：直接把 DB 层的完整行返回给上层（external handler）
func GetSlackConfigUI() (dbcfg.SlackConfigRow, error) {
	return dbcfg.GetSlackConfigFull()
}

// Web 更新入参（全部可选，nil 表示不更新该字段）
type SlackUpdateReq struct {
	Enable      *int   `json:"enable,omitempty"`       // 0/1
	Webhook     *string `json:"webhook,omitempty"`     // 为空串也会写入
	IntervalSec *int64 `json:"intervalSec,omitempty"`  // 秒
}

// UI 层更新：不做业务逻辑，直接转调 DB 层
func UpdateSlackConfigUI(req SlackUpdateReq) error {
	return dbcfg.UpdateSlackConfig(dbcfg.SlackConfigUpdate{
		Enable:      req.Enable,
		Webhook:     req.Webhook,
		IntervalSec: req.IntervalSec,
	})
}
