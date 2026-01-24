package gateway

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"AtlHyper/atlhyper_agent_v2/model"
	"AtlHyper/model_v2"
)

// masterGateway Master 通信实现
//
// 使用 HTTP 协议与 Master 通信:
//   - 快照推送使用 Gzip 压缩 (减少带宽)
//   - 指令拉取使用长轮询 (减少请求频率)
//   - 所有请求带 X-Cluster-ID 头标识集群
type masterGateway struct {
	masterURL  string       // Master 服务地址
	clusterID  string       // 集群标识
	httpClient *http.Client // HTTP 客户端 (复用连接)
}

// NewMasterGateway 创建 Master 网关
//
// 参数:
//   - masterURL: Master 服务地址，如 "http://master:8080"
//   - clusterID: 集群标识
//   - httpTimeout: HTTP 客户端超时时间 (长轮询需要较长超时)
func NewMasterGateway(masterURL, clusterID string, httpTimeout time.Duration) MasterGateway {
	return &masterGateway{
		masterURL: masterURL,
		clusterID: clusterID,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// PushSnapshot 推送快照
func (g *masterGateway) PushSnapshot(ctx context.Context, snapshot *model_v2.ClusterSnapshot) error {
	// 1. JSON 序列化
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// 2. Gzip 压缩 (快照数据较大，压缩可显著减少带宽)
	compressed, err := g.gzipCompress(data)
	if err != nil {
		return fmt.Errorf("failed to compress snapshot: %w", err)
	}

	// 3. 构建请求
	url := fmt.Sprintf("%s/agent/snapshot", g.masterURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Cluster-ID", g.clusterID)

	// 4. 发送请求
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// commandResponse Master 返回的指令响应格式
type commandResponse struct {
	HasCommand bool         `json:"has_command"`
	Command    *commandInfo `json:"command,omitempty"`
}

// commandInfo 指令信息（与 Master agentsdk.CommandInfo 对应）
type commandInfo struct {
	ID              string                 `json:"id"`
	Action          string                 `json:"action"`
	TargetKind      string                 `json:"target_kind"`
	TargetNamespace string                 `json:"target_namespace"`
	TargetName      string                 `json:"target_name"`
	Params          map[string]interface{} `json:"params,omitempty"`
}

// PollCommands 拉取指令 (长轮询)
func (g *masterGateway) PollCommands(ctx context.Context, topic string) ([]model.Command, error) {
	url := fmt.Sprintf("%s/agent/commands?cluster_id=%s&topic=%s", g.masterURL, g.clusterID, topic)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Cluster-ID", g.clusterID)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		// 超时或取消是正常的长轮询行为，不返回错误
		if ctx.Err() != nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 204 表示没有指令
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析 Master 返回的格式
	var cmdResp commandResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmdResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 没有指令
	if !cmdResp.HasCommand || cmdResp.Command == nil {
		return nil, nil
	}

	// 转换为 model.Command
	cmd := model.Command{
		ID:        cmdResp.Command.ID,
		Action:    cmdResp.Command.Action,
		Kind:      cmdResp.Command.TargetKind,
		Namespace: cmdResp.Command.TargetNamespace,
		Name:      cmdResp.Command.TargetName,
		Params:    cmdResp.Command.Params,
	}

	return []model.Command{cmd}, nil
}

// ReportResult 上报执行结果
func (g *masterGateway) ReportResult(ctx context.Context, result *model.Result) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	url := fmt.Sprintf("%s/agent/result", g.masterURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cluster-ID", g.clusterID)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Heartbeat 心跳
func (g *masterGateway) Heartbeat(ctx context.Context) error {
	url := fmt.Sprintf("%s/agent/heartbeat", g.masterURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Cluster-ID", g.clusterID)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}

// gzipCompress 压缩数据
func (g *masterGateway) gzipCompress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
