package config

import "testing"

func TestServerConfig_Location(t *testing.T) {
	if loc, err := (ServerConfig{}).Location(); err != nil || loc.String() != "UTC" {
		t.Errorf("没配 = (%v, %v), want UTC", loc, err)
	}
	if loc, err := (ServerConfig{Timezone: " Asia/Tokyo "}).Location(); err != nil || loc.String() != "Asia/Tokyo" {
		t.Errorf("Asia/Tokyo = (%v, %v)", loc, err)
	}
	// 写错不静默退回 UTC：启动时报错
	if _, err := (ServerConfig{Timezone: "Not/AZone"}).Location(); err == nil {
		t.Error("不合法的时区名应返回错误")
	}
}
