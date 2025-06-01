// =======================================================================================
// 📄 mapper.go
//
// ✨ 功能说明：
//     本模块负责维护 tag → Deployment 的映射逻辑，
//     用于 webhook 接收到镜像推送事件后，确定对应的 Kubernetes 资源。
//     映射配置来自 config/webhook_map.yaml，可动态加载和查询。
//
// 🛠️ 提供功能：
//     - LoadTagDeploymentMap(): 加载 YAML 格式的映射文件
//     - GetDeploymentForTag(tag string): 获取 tag 对应的 Deployment 名称
//
// 📦 依赖：
//     - gopkg.in/yaml.v2
//     - 内部模块：logger
//
// 📍 使用场景：
//     - 被 webhook handler 调用，根据 tag 查找部署目标
//     - 可用于未来扩展的自动化发布策略（如 Flagger）
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package webhook
