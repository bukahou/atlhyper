package events

import (
	"NeuroController/db/repository/eventlog"
	"NeuroController/model"
	"time"
)

func GetRecentEventLogs(clusterID string, withinDays int) ([]model.EventLog, error) {
	// 构造起始时间戳：当前时间 - N 天
	since := time.Now().
		Add(-time.Duration(withinDays) * 24 * time.Hour).
		Format(time.RFC3339)

	// 调用底层持久层查询函数
	logs, err := eventlog.GetEventLogsSince(clusterID, since)
	if err != nil {
		return nil, err
	}
	
	return logs, nil
}
