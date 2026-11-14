package slo

import (
	"strings"
	"testing"

	"AtlHyper/atlhyper_master_v2/model"
)

func TestValidateTargetRequest(t *testing.T) {
	valid := func() model.UpdateSLOTargetRequest {
		return model.UpdateSLOTargetRequest{Host: "example.com", WindowDays: 7, AvailabilityTarget: 99.9, P95LatencyTarget: 300}
	}
	cases := []struct {
		name    string
		mutate  func(*model.UpdateSLOTargetRequest)
		wantErr bool
	}{
		{"合法", func(*model.UpdateSLOTargetRequest) {}, false},
		{"窗口为 0 走默认值", func(r *model.UpdateSLOTargetRequest) { r.WindowDays = 0 }, false},
		{"窗口 1 天", func(r *model.UpdateSLOTargetRequest) { r.WindowDays = 1 }, false},
		{"通配域名", func(r *model.UpdateSLOTargetRequest) { r.Host = "*.example.com" }, false},
		{"大写域名", func(r *model.UpdateSLOTargetRequest) { r.Host = "API.Example.com" }, false},
		{"可用率 100", func(r *model.UpdateSLOTargetRequest) { r.AvailabilityTarget = 100 }, false},
		{"缺 host", func(r *model.UpdateSLOTargetRequest) { r.Host = "" }, true},
		{"host 含空格", func(r *model.UpdateSLOTargetRequest) { r.Host = "a b.com" }, true},
		{"host 含路径", func(r *model.UpdateSLOTargetRequest) { r.Host = "example.com/x" }, true},
		{"host 超长", func(r *model.UpdateSLOTargetRequest) { r.Host = strings.Repeat("a.", 127) + "com" }, true},
		{"窗口不在选项内", func(r *model.UpdateSLOTargetRequest) { r.WindowDays = 30 }, true},
		{"窗口为负", func(r *model.UpdateSLOTargetRequest) { r.WindowDays = -1 }, true},
		{"可用率为 0", func(r *model.UpdateSLOTargetRequest) { r.AvailabilityTarget = 0 }, true},
		{"可用率超过 100", func(r *model.UpdateSLOTargetRequest) { r.AvailabilityTarget = 100.1 }, true},
		{"延迟为 0", func(r *model.UpdateSLOTargetRequest) { r.P95LatencyTarget = 0 }, true},
		{"延迟超过 10 分钟", func(r *model.UpdateSLOTargetRequest) { r.P95LatencyTarget = 600_001 }, true},
		{"集群 ID 超长", func(r *model.UpdateSLOTargetRequest) { r.ClusterID = strings.Repeat("c", 129) }, true},
	}
	for _, c := range cases {
		req := valid()
		c.mutate(&req)
		err := validateTargetRequest(&req)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", c.name, err, c.wantErr)
		}
	}
}
