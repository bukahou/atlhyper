package operations

import (
	"context"
	"errors"
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
)

func smtpUpdateRequest(password string) *model.UpdateNotifySMTPRequest {
	return &model.UpdateNotifySMTPRequest{Host: " smtp.example.com ", Port: 587, User: "ops@example.com",
		Password: password, TLS: true, FromAddress: ""}
}

func TestUpdateNotifySMTP_FirstSaveRequiresPassword(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()
	if err := svc.UpdateNotifySMTP(ctx, smtpUpdateRequest("")); !errors.Is(err, model.ErrInvalidNotifyInput) {
		t.Fatalf("第一次保存不带密码应被拒绝，实际 %v", err)
	}
	if stored, _ := db.NotifySMTP.Get(ctx); stored != nil {
		t.Fatalf("被拒绝时不应落库: %+v", stored)
	}

	if err := svc.UpdateNotifySMTP(ctx, smtpUpdateRequest(notifyTestPassword)); err != nil {
		t.Fatal(err)
	}
	stored, _ := db.NotifySMTP.Get(ctx)
	if stored == nil || stored.Host != "smtp.example.com" || stored.Password != notifyTestPassword || !stored.Configured() {
		t.Fatalf("保存的设置不对（主机应去掉首尾空白）: host=%q configured=%v", stored.Host, stored.Configured())
	}
}

// 密码留空 = 保持原值；其余字段以请求为准；给了新密码则替换
func TestUpdateNotifySMTP_BlankPasswordKeepsStoredValue(t *testing.T) {
	svc, db := newNotifyTestService(t)
	ctx := context.Background()
	if err := svc.UpdateNotifySMTP(ctx, smtpUpdateRequest(notifyTestPassword)); err != nil {
		t.Fatal(err)
	}

	update := smtpUpdateRequest("")
	update.Port, update.FromAddress = 465, "alerts@example.com"
	if err := svc.UpdateNotifySMTP(ctx, update); err != nil {
		t.Fatal(err)
	}
	stored, _ := db.NotifySMTP.Get(ctx)
	if stored.Port != 465 || stored.FromAddress != "alerts@example.com" || stored.Password != notifyTestPassword {
		t.Errorf("密码留空应保持原值、其余字段应更新: port=%d from=%s", stored.Port, stored.FromAddress)
	}

	if err := svc.UpdateNotifySMTP(ctx, smtpUpdateRequest(notifyTestPassword+"-rotated")); err != nil {
		t.Fatal(err)
	}
	stored, _ = db.NotifySMTP.Get(ctx)
	if stored.Password != notifyTestPassword+"-rotated" {
		t.Error("给了新密码应替换")
	}
}

func TestUpdateNotifySMTP_InvalidInput(t *testing.T) {
	svc, _ := newNotifyTestService(t)
	bad := smtpUpdateRequest(notifyTestPassword)
	bad.Port = 0
	if err := svc.UpdateNotifySMTP(context.Background(), bad); !errors.Is(err, model.ErrInvalidNotifyInput) {
		t.Fatalf("端口为 0 应被拒绝，实际 %v", err)
	}
}
