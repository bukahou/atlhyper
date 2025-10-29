// atlhyper_aiservice/client/master/master_client.go
package master

import (
	"AtlHyper/atlhyper_aiservice/config"
	model "AtlHyper/model/ai"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
)

// ============================================================
// 🧠 Master API 客户端
// ------------------------------------------------------------
// - 提供通用的 Master 调用函数
// - 各 API Path 由配置或常量定义
// ============================================================

// 常量定义所有可用的 Master 接口路径
const (
	PathFetchContext = "/ai/context/fetch" // 拉取集群资源上下文
)

// doPost —— 通用 POST 请求方法（用于 Master 调用）
func doPost[T any](ctx context.Context, endpoint string, reqBody any) (*T, error) {
	cfg := config.GetMasterAPI()
	url := fmt.Sprintf("%s%s", cfg.BaseURL, path.Clean(endpoint))

	body, _ := json.Marshal(reqBody)
	httpClient := &http.Client{Timeout: cfg.Timeout}

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("call master failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("master non-200: %d %s", resp.StatusCode, string(b))
	}

	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode master resp failed: %w", err)
	}
	return &out, nil
}

// ============================================================
// 🧩 业务层封装（具体 API 调用）
// ============================================================

// FetchAIContext —— 拉取集群上下文资源（Pod/Deployment/Node/...）
func FetchAIContext(ctx context.Context, req *model.AIFetchRequest) (*model.AIFetchResponse, error) {
	return doPost[model.AIFetchResponse](ctx, PathFetchContext, req)
}
