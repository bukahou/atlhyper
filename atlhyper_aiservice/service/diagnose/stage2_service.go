// atlhyper_aiservice/service/diagnose/stage2_service.go
package diagnose

import (
	"AtlHyper/atlhyper_aiservice/client/master"
	model "AtlHyper/model/ai"
	"context"
	"fmt"
)

// RunStage3FetchContext —— 阶段2b：调用 Master 获取上下文
// --------------------------------------------------------------
// 📘 功能：
//   - 接收 Stage2a 解析得到的 AIFetchRequest。
//   - 调用 Master /ai/context/fetch 获取上下文详情。
//   - 返回 fetch 结果。
func RunStage3FetchContext(ctx context.Context, req *model.AIFetchRequest) (map[string]interface{}, error) {
	resp, err := master.FetchAIContext(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("fetch context failed: %v", err)
	}

	return map[string]interface{}{
		"need":  req,
		"fetch": resp,
	}, nil
}
