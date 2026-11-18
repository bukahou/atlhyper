// atlhyper_master_v2/service/operations/notify_smtp.go
// OpsNotify 实现 — 全局发件设置的写入
//
// ⛔ SMTP 密码不写日志；校验错误的文案不回显它
package operations

import (
	"context"
	"strings"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
)

// UpdateNotifySMTP 整体替换全局发件设置。
//
// 密码留空 = 保持库里的原值；库里还没有密码时必须填写。输入不合规 → model.ErrInvalidNotifyInput
func (s *NotifyService) UpdateNotifySMTP(ctx context.Context, req *model.UpdateNotifySMTPRequest) error {
	existing, err := s.repos.SMTP.Get(ctx)
	if err != nil {
		return err
	}
	storedPassword := ""
	if existing != nil {
		storedPassword = existing.Password
	}

	trimmed := *req
	trimmed.Host, trimmed.User, trimmed.FromAddress = strings.TrimSpace(req.Host), strings.TrimSpace(req.User), strings.TrimSpace(req.FromAddress)
	if err := model.ValidateNotifySMTP(&trimmed, storedPassword == ""); err != nil {
		return err
	}

	password := trimmed.Password
	if password == "" {
		password = storedPassword
	}
	settings := &database.NotifySMTP{
		Host: trimmed.Host, Port: trimmed.Port, User: trimmed.User, Password: password,
		TLS: trimmed.TLS, FromAddress: trimmed.FromAddress,
	}
	if err := s.repos.SMTP.Save(ctx, settings); err != nil {
		return err
	}
	notifyLog.Info("发件设置已保存", "host", settings.Host, "port", settings.Port, "passwordChanged", trimmed.Password != "")
	return nil
}
