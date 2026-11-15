package repo_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
)

func TestNotifySMTP_NilWhenNeverSaved(t *testing.T) {
	db := openNotifyTestDB(t)
	settings, err := db.NotifySMTP.Get(context.Background())
	if err != nil || settings != nil {
		t.Fatalf("从没保存过应返回 (nil, nil)，实际 (%+v, %v)", settings, err)
	}
}

// 整份配置加密落库：config 列里不能有密码、主机、账号的明文；经仓库读出时已解密；再存一次仍是同一行
func TestNotifySMTP_SaveEncryptedThenGet(t *testing.T) {
	db := openNotifyTestDB(t)
	ctx := context.Background()
	settings := &database.NotifySMTP{
		Host: "smtp.example.com", Port: 587, User: "ops@example.com",
		Password: testSMTPPassword, TLS: true, FromAddress: "alerts@example.com",
	}
	if err := db.NotifySMTP.Save(ctx, settings); err != nil {
		t.Fatal(err)
	}
	if settings.UpdatedAt.IsZero() {
		t.Error("保存后应回填 UpdatedAt")
	}
	settings.Port = 465
	if err := db.NotifySMTP.Save(ctx, settings); err != nil {
		t.Fatal(err)
	}

	rows, err := db.Conn.Query("SELECT config FROM notify_smtp")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var stored string
		if err := rows.Scan(&stored); err != nil {
			t.Fatal(err)
		}
		count++
		for _, plain := range []string{testSMTPPassword, "smtp.example.com", "ops@example.com", "password"} {
			if strings.Contains(stored, plain) {
				t.Errorf("config 列里出现了明文 %q", plain)
			}
		}
		if !strings.HasPrefix(stored, "enc:") {
			t.Errorf("config 列应是密文，实际 %q", stored)
		}
	}
	if count != 1 {
		t.Fatalf("单行表应只有 1 行，实际 %d", count)
	}

	got, err := db.NotifySMTP.Get(ctx)
	if err != nil || got == nil {
		t.Fatalf("应读到设置，实际 (%+v, %v)", got, err)
	}
	if got.Host != "smtp.example.com" || got.Port != 465 || got.User != "ops@example.com" ||
		got.Password != testSMTPPassword || !got.TLS || got.FromAddress != "alerts@example.com" || got.UpdatedAt.IsZero() {
		t.Errorf("读出的设置不对: host=%s port=%d user=%s tls=%v from=%s", got.Host, got.Port, got.User, got.TLS, got.FromAddress)
	}
	if !got.Configured() {
		t.Error("四项都有时 Configured 应为真")
	}
}

func TestNotifySMTP_Configured(t *testing.T) {
	full := database.NotifySMTP{Host: "h", Port: 587, User: "u", Password: "p"}
	cases := map[string]*database.NotifySMTP{
		"nil": nil, "缺主机": {Port: 587, User: "u", Password: "p"}, "缺端口": {Host: "h", User: "u", Password: "p"},
		"缺账号": {Host: "h", Port: 587, Password: "p"}, "缺密码": {Host: "h", Port: 587, User: "u"},
	}
	for name, settings := range cases {
		if settings.Configured() {
			t.Errorf("%s: Configured 应为假", name)
		}
	}
	if !full.Configured() {
		t.Error("四项齐全时 Configured 应为真")
	}
}

func TestNotifySMTP_SaveRefusedWithoutEncryptor(t *testing.T) {
	db := openNotifyTestDB(t)
	restore := repo.DisableEncryptorForTest()
	defer restore()
	err := db.NotifySMTP.Save(context.Background(), &database.NotifySMTP{Host: "h", Port: 587, User: "u", Password: "p"})
	if !errors.Is(err, database.ErrNotifyEncryptionUnavailable) {
		t.Fatalf("加密器不可用时应拒绝保存，实际 %v", err)
	}
}
