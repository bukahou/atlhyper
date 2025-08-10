package metrics

// NetworkStat 表示节点某个网卡接口的实时网络流量统计（单位：KB/s 和可读格式）
type NetworkStat struct {
	Interface string  `json:"interface"`      // 网络接口名称，如 eth0, wlan0
	RxKBps    float64 `json:"rxKBps"`         // 接收速率（KB/s）
	TxKBps    float64 `json:"txKBps"`         // 发送速率（KB/s）
	RxSpeed   string  `json:"rxSpeed"`        // 接收速率（如 1.25 MB/s）
	TxSpeed   string  `json:"txSpeed"`        // 发送速率（如 982 KB/s）
}
