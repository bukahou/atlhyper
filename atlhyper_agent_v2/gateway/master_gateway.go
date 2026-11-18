package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"AtlHyper/common"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// masterGateway Master 通信实现
//
// 使用 HTTP 协议与 Master 通信:
//   - 快照推送使用 Gzip 压缩 (减少带宽)
//   - 指令拉取使用长轮询 (减少请求频率)
//   - 所有请求带 X-Cluster-ID 头标识集群
//   - 配置了集群令牌时，所有请求带 Authorization: Bearer 头证明集群身份
type masterGateway struct {
	masterURL   string       // Master 服务地址
	clusterID   string       // 集群标识
	masterToken string       // 集群令牌；空 = 不带认证头。⛔ 不得写入日志或错误信息
	httpClient  *http.Client // HTTP 客户端 (复用连接)
}

// maxErrorBodyBytes 写进错误信息的响应体上限：只取开头一小段说明原因，不把整段内容带进日志
const maxErrorBodyBytes = 256

// NewMasterGateway 创建 Master 网关
//
// 参数:
//   - masterURL: Master 服务地址，如 "http://master:8080"
//   - clusterID: 集群标识
//   - masterToken: 集群令牌（Master 签发）；空串表示不带认证头
//   - httpTimeout: HTTP 客户端超时时间 (长轮询需要较长超时)
func NewMasterGateway(masterURL, clusterID, masterToken string, httpTimeout time.Duration) MasterGateway {
	return &masterGateway{
		masterURL:   masterURL,
		clusterID:   clusterID,
		masterToken: masterToken,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// newRequest 构建发往 Master 的请求：四条通道共用，保证集群标识与认证头一处设置、不会漏掉
func (g *masterGateway) newRequest(ctx context.Context, method, path string, query url.Values, body io.Reader) (*http.Request, error) {
	target := g.masterURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-Cluster-ID", g.clusterID)
	if g.masterToken != "" {
		req.Header.Set("Authorization", "Bearer "+g.masterToken)
	}
	return req, nil
}

// unexpectedStatusError 把非预期的状态码转成错误。
// 401 / 403 包成 ErrMasterRejectedIdentity，让上层与日志能认出这是身份问题而不是网络问题。
func unexpectedStatusError(resp *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	reason := strings.TrimSpace(string(body))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("%w: HTTP %d, body: %s", ErrMasterRejectedIdentity, resp.StatusCode, reason)
	}
	return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, reason)
}

// PushSnapshot 推送快照
func (g *masterGateway) PushSnapshot(ctx context.Context, snapshot *cluster.ClusterSnapshot) error {
	// 1. JSON 序列化
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// 2. Gzip 压缩 (快照数据较大，压缩可显著减少带宽)
	compressed, err := common.GzipBytes(data)
	if err != nil {
		return fmt.Errorf("failed to compress snapshot: %w", err)
	}

	// 3. 构建请求
	req, err := g.newRequest(ctx, http.MethodPost, "/agent/snapshot", nil, bytes.NewReader(compressed))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")

	// 4. 发送请求
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return unexpectedStatusError(resp)
	}

	return nil
}

// commandResponse Master 返回的指令响应格式
// 直接使用 command.Command，无需中间类型转换
// JSON tag 必须与 Master agentsdk/types.go 中的 CommandResponse 一致（camelCase）
type commandResponse struct {
	HasCommand bool             `json:"hasCommand"`
	Command    *command.Command `json:"command,omitempty"`
}

// PollCommands 拉取指令 (长轮询)
func (g *masterGateway) PollCommands(ctx context.Context, topic string) ([]command.Command, error) {
	// 集群 ID 与 topic 经 url.Values 转义：取值里的 & # 空格等字符不会改变请求的含义
	query := url.Values{"cluster_id": {g.clusterID}, "topic": {topic}}
	req, err := g.newRequest(ctx, http.MethodGet, "/agent/commands", query, nil)
	if err != nil {
		return nil, err
	}

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
		return nil, unexpectedStatusError(resp)
	}

	// 解析 Master 返回的格式，直接使用 command.Command
	var cmdResp commandResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmdResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 没有指令
	if !cmdResp.HasCommand || cmdResp.Command == nil {
		return nil, nil
	}

	// 直接返回，无需转换
	return []command.Command{*cmdResp.Command}, nil
}

// ReportResult 上报执行结果
func (g *masterGateway) ReportResult(ctx context.Context, result *command.Result) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	req, err := g.newRequest(ctx, http.MethodPost, "/agent/result", nil, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return unexpectedStatusError(resp)
	}

	return nil
}

// Heartbeat 心跳
func (g *masterGateway) Heartbeat(ctx context.Context) error {
	req, err := g.newRequest(ctx, http.MethodPost, "/agent/heartbeat", nil, nil)
	if err != nil {
		return err
	}

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return unexpectedStatusError(resp)
	}

	return nil
}
