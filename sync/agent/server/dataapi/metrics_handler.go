// sync/agent/server/dataapi/handlers.go
package dataapi

// import (
// 	"net/http"
// 	"time"

// 	iface "NeuroController/internal/interfaces/data_api" // 你放 interfaces 的包名路径

// 	"github.com/gin-gonic/gin"
// )

// type MetricsHandlers struct {
// 	api *iface.MetricsStoreAPI
// }

// func NewMetricsHandlers(api *iface.MetricsStoreAPI) *MetricsHandlers {
// 	return &MetricsHandlers{api: api}
// }

// // GET /agent/dataapi/metrics/all
// // 返回当前 store 中所有节点的全部快照（测试/调试用；生产慎用）
// func (h *MetricsHandlers) GetAll(c *gin.Context) {
// 	data := h.api.DumpAll()
// 	c.JSON(http.StatusOK, data)
// }

// // GetLatest 处理“获取最新快照”的请求。
// // 行为：
// //  - ?node=<name> 存在：仅返回该节点的最新快照（以 map 形式返回，key 为节点名）。
// //  - 未指定 node：返回所有节点各自的最新快照（map[节点名]最新快照）。
// func (h *MetricsHandlers) GetLatest(c *gin.Context) {
//     // 读取查询参数：可选的节点名称
//     node := c.Query("node")

//     // 用于承载响应数据；选择 map 是为了让前端按节点名直接索引
//     result := make(map[string]interface{})

//     // 分支一：如果指定了某个节点
//     if node != "" {
//         // 通过 interfaces 层拿到该节点的最新快照
//         if latest := h.api.GetLatest(node); latest != nil {
//             // 如果有数据，就以 <node>: <snapshot> 的形式放入结果
//             result[node] = latest
//         }
//         // 返回 JSON。注意：如果 latest 为 nil，这里会返回空对象 {}
//         c.JSON(http.StatusOK, result)
//         return
//     }

//     // 分支二：未指定节点 —— 返回所有节点的“各自最新一条”
//     // 从接口层获取所有节点的历史快照副本（深拷贝，线程安全）
//     all := h.api.DumpAll()

//     // 遍历每个节点的切片，取切片末尾（最新）塞入结果
//     for n, arr := range all {
//         if len(arr) > 0 {
//             result[n] = arr[len(arr)-1] // 末尾即最新
//         }
//         // 若某节点 arr 为空则不写入，让结果保持“只包含有数据的节点”
//     }

//     // 返回形如：{"desk-drei": {...最新...}, "desk-zwei": {...最新...}}
//     c.JSON(http.StatusOK, result)
// }

// // GetRange 处理 GET /xxx/range 请求，返回指定节点在给定时间窗内的所有监控快照
// func (h *MetricsHandlers) GetRange(c *gin.Context) {
//     // 从 URL 查询参数中获取 node 名称，例如 /range?node=desk-zwei
//     node := c.Query("node")
//     if node == "" {
//         // 如果没有指定 node 参数，返回 HTTP 400 错误
//         c.JSON(http.StatusBadRequest, gin.H{"error": "missing node"})
//         return
//     }

//     // 从 URL 查询参数中解析时间窗参数 since 和 until
//     // 如果参数不存在或格式错误，将使用默认值：最近 5 分钟
//     since, until := parseWindow(c.Query("since"), c.Query("until"))

//     // 调用 API 层的 Range 方法，获取该节点在时间窗内的所有快照数据
//     list := h.api.Range(node, since, until)

//     // 以 JSON 格式返回结果
//     c.JSON(http.StatusOK, list)
// }

// // parseWindow 解析时间窗参数
// // s1 表示起始时间 since，s2 表示结束时间 until
// // 时间字符串必须为 RFC3339 格式，例如 "2025-08-11T12:00:00Z"
// // 如果参数为空或解析失败，则使用默认时间窗：当前时间前 5 分钟到当前时间
// func parseWindow(s1, s2 string) (time.Time, time.Time) {
//     now := time.Now()
//     // 默认时间窗：从当前时间往前 5 分钟，到当前时间
//     since, until := now.Add(-5*time.Minute), now

//     // 如果 since 参数存在且能解析成功，则覆盖默认 since
//     if t, err := time.Parse(time.RFC3339, s1); err == nil {
//         since = t
//     }

//     // 如果 until 参数存在且能解析成功，则覆盖默认 until
//     if t, err := time.Parse(time.RFC3339, s2); err == nil {
//         until = t
//     }

//     // 返回解析后的时间窗
//     return since, until
// }
