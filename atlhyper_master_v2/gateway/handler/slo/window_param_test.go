package slo

import (
	"net/http/httptest"
	"testing"

	slomodel "AtlHyper/model_v3/slo"
)

func TestSLOWindowParam(t *testing.T) {
	cases := []struct {
		query   string
		want    string
		wantErr bool
	}{
		{"", slomodel.DefaultWindow, false},
		{"time_range=7d", "7d", false},
		{"time_range=1d", "", true}, // 不是预先算好的窗口：报错，不悄悄换成实时数据
		{"time_range=30d", "", true},
	}
	for _, c := range cases {
		got, err := sloWindowParam(httptest.NewRequest("GET", "/x?"+c.query, nil))
		if (err != nil) != c.wantErr || got != c.want {
			t.Errorf("%q → (%q, %v), want (%q, err=%v)", c.query, got, err, c.want, c.wantErr)
		}
	}
}
