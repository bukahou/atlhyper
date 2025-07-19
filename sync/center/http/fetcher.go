package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// AgentEndpoints 是 Agent 节点的地址列表（可改为读取配置）
// var AgentEndpoints = []string{
// 	"http://127.0.0.1:18080",
// }

// AgentEndpoints 是从环境变量读取的 Agent 节点地址列表（默认值为 localhost）
var AgentEndpoints = loadAgentEndpoints()

// loadAgentEndpoints 从环境变量读取 AGENT_ENDPOINTS（多个地址用英文逗号 , 分隔）
func loadAgentEndpoints() []string {
	env := os.Getenv("AGENT_ENDPOINTS")
	if env == "" {
		return []string{"http://127.0.0.1:8082"}
	}
	endpoints := strings.Split(env, ",")
	for i := range endpoints {
		endpoints[i] = strings.TrimSpace(endpoints[i])
	}
	return endpoints
}

// ===============================
// ✅ 通用 HTTP 请求函数
// ===============================


// GET 工具（仅第一个 Agent）
func GetFromAgent[T any](path string, result *T) error {
	client := &http.Client{Timeout: 5 * time.Second}
	url := AgentEndpoints[0] + path // ❗让调用方传入完整路径

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("❌ 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("❌ 响应失败 (%d): %s", resp.StatusCode, string(body))
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(result); err != nil {
		return fmt.Errorf("❌ JSON 解析失败: %w", err)
	}
	return nil
}


func PostToAgent(path string, payload any) error {
	client := &http.Client{Timeout: 5 * time.Second}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("❌ JSON 编码失败: %w", err)
	}

	for _, base := range AgentEndpoints {
		url := base + path  // ❗ 直接拼接 path，外部传入完整路径
		resp, err := client.Post(url, "application/json", bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("❌ 请求 %s 失败: %w", url, err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			data, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("❌ Agent 响应失败 (%d): %s", resp.StatusCode, string(data))
		}
	}
	return nil
}

// GET 工具：返回纯文本（string）
func GetTextFromAgent(path string, result *string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	url := AgentEndpoints[0] + path

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("❌ 请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("❌ 响应失败 (%d): %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("❌ 读取响应失败: %w", err)
	}

	*result = string(body)
	return nil
}

// POST 工具：发送 application/x-www-form-urlencoded
func PostFormToAgent(path string, form url.Values, result any) error {
	client := &http.Client{Timeout: 5 * time.Second}
	url := AgentEndpoints[0] + path

	resp, err := client.Post(
		url,
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("❌ 表单请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("❌ 表单响应失败 (%d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		decoder := json.NewDecoder(resp.Body)
		if err := decoder.Decode(result); err != nil {
			return fmt.Errorf("❌ JSON 解析失败: %w", err)
		}
	}

	return nil
}

