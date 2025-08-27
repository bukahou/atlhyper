package master_metrics

// GetLatestNodeMetrics 从 agent 获取所有节点的最新指标数据
// func GetLatestNodeMetrics() (json.RawMessage, error) {
//     var raw json.RawMessage
//     err := http.GetFromAgent("/agent/dataapi/latest", &raw)
//     return raw, err
// }

// func saveLatestSnapshotsOnce() error {
//     ctx := context.Background()

//     // 直接按对象形状解码
//     var obj map[string]*model.NodeMetricsSnapshot
//     if err := http.GetFromAgent("/agent/dataapi/latest", &obj); err != nil {
//         return err
//     }

//     // 转换成 map[string][]*
//     arr := make(map[string][]*model.NodeMetricsSnapshot, len(obj))
//     for k, v := range obj {
//         if v == nil {
//             continue
//         }
//         arr[k] = []*model.NodeMetricsSnapshot{v}
//     }

//     return dbmetrics.UpsertSnapshots(ctx, utils.DB, arr)
// }
