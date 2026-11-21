package observe

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"AtlHyper/api/contract"
	"AtlHyper/model_v3/command"
)

// 失败类别以 OpenAPI 规格为准（前端类型由规格生成）：Go 常量与规格里的枚举逐个对上
func TestObserveFailureCodes_MatchSpec(t *testing.T) {
	doc, err := contract.Spec()
	if err != nil {
		t.Fatal(err)
	}
	ref, ok := doc.Components.Schemas["ObserveFailureCode"]
	if !ok {
		t.Fatal("规格里没有 ObserveFailureCode")
	}
	var specCodes []string
	for _, v := range ref.Value.Enum {
		specCodes = append(specCodes, v.(string))
	}
	goCodes := []string{command.CodeNotConfigured, command.CodeInvalidParams, command.CodeTimeout,
		command.CodeUnavailable, command.CodeRejected, command.CodeFailed}
	slices.Sort(specCodes)
	slices.Sort(goCodes)
	if !slices.Equal(specCodes, goCodes) {
		t.Fatalf("规格 %v ≠ Go %v", specCodes, goCodes)
	}
	for _, code := range goCodes {
		if !command.IsKnownResultCode(code) {
			t.Errorf("IsKnownResultCode(%q) = false", code)
		}
	}
}

// 观测查询失败的响应（Agent 失败、参数不合法）都符合规格里的 ObserveFailureResponse
func TestWriteQueryError_ResponseMatchesSpec(t *testing.T) {
	doc, err := contract.Spec()
	if err != nil {
		t.Fatal(err)
	}
	schema := doc.Components.Schemas["ObserveFailureResponse"].Value

	agentFailure := newCommandQuery(&blockingOps{result: &command.Result{Success: false, Code: command.CodeTimeout, Error: "raw"}})
	_, agentErr := agentFailure.run(context.Background(), "c1", command.ActionQueryLogs, command.QueryLogsParams{}, time.Minute)
	_, invalidErr := newCommandQuery(&blockingOps{}).run(context.Background(), "c1", command.ActionQueryLogs, nil, time.Minute)

	for name, err := range map[string]error{"Agent 失败": agentErr, "参数不合法": invalidErr} {
		rec := httptest.NewRecorder()
		writeQueryError(rec, err)
		var body any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: 响应不是 JSON: %v", name, err)
		}
		if err := schema.VisitJSON(body); err != nil {
			t.Errorf("%s: 不符合 ObserveFailureResponse: %v（%s）", name, err, rec.Body.String())
		}
	}
}
