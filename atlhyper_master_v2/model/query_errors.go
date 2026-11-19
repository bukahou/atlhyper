package model

import "errors"

// ErrSnapshotNotReady 集群还没有上报过快照（刚接入或 Agent 未连上）。与「查询失败」区分，网关返回 404
var ErrSnapshotNotReady = errors.New("数据尚未就绪")
