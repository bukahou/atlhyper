// =======================================================================================
// 📄 handler.go
//
// ✨ 功能说明：
//     Webhook 模块的入口处理器，用于接收来自 DockerHub 的镜像推送事件，
//     解析其中的 tag、repo 等信息，并根据 mapper 配置匹配目标 Deployment，
//     触发相应的策略操作（如重启、标记更新、未来联动灰度发布等）。
//
// 🛠️ 提供功能：
//     - HandleDockerHubWebhook(): Gin 路由绑定的主要处理函数
//     - 解析 JSON Payload，提取 tag、repo、pusher 等字段
//     - 调用 mapper 模块获取对应的 deployment
//
// 📦 依赖：
//     - Gin Web 框架（github.com/gin-gonic/gin）
//     - 内部模块：mapper、logger
//
// 📍 使用场景：
//     - 接收 DockerHub webhook 推送
//     - 用于自动化部署触发入口
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package webhook
