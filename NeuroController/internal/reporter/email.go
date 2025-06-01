// =======================================================================================
// 📄 email.go
//
// ✨ 功能说明：
//     本模块封装邮件报警功能，基于 SMTP 协议发送 HTML 格式的告警邮件。
//     支持配置发件人、收件人、SMTP 服务地址等参数，
//     并提供统一接口供其他模块（如 watcher、trigger）调用告警逻辑。
//
// 🛠️ 提供功能：
//     - SendAlertEmail(subject string, htmlBody string): 发送 HTML 格式的告警邮件
//
// 📦 依赖：
//     - net/smtp（标准库）
//     - config 模块：用于获取 SMTP 配置项
//
// 📍 使用场景：
//     - Pod 异常触发报警（由 watcher 模块调用）
//     - AI 模块识别高风险错误后通知人工干预
//
// 📫 配置项：
//     读取自 config/config.yaml，例如：
//     ```yaml
//     email:
//       smtp_server: smtp.example.com
//       sender: bot@example.com
//       password: your-password
//       to: alert@example.com
//     ```
//
// ✍️ 作者：武夏锋（@ZGMF-X10A）
// 📅 创建时间：2025-06
// =======================================================================================

package reporter
