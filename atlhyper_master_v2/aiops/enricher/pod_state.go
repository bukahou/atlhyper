// atlhyper_master_v2/aiops/enricher/pod_state.go
// 后台分析上下文：事件涉及 Pod 的容器状态（A10）
//
// 后台分析不能调工具，只看得到风险与时间线，原因只能写「未知（证据不足）」。
// Agent 快照里本就有每个容器的当前状态与上次终止信息 —— 带进上下文，零额外查询。
// ⛔ 不带日志：日志要额外查 ClickHouse（dev / prod 共用），每次自动分析都多一次查询。
package enricher

import (
	"fmt"
	"strings"

	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/model_v3/cluster"
)

const (
	maxPodStateEntries     = 5   // 最多带几个 Pod
	maxTerminationMsgChars = 200 // 终止信息截断长度（可能含栈 / 配置内容）
)

// buildPodStateContext 事件中 Pod 类实体的容器状态；无快照或找不到 Pod 时返回空串
func buildPodStateContext(snap *cluster.ClusterSnapshot, entities []*database.AIOpsIncidentEntity) string {
	if snap == nil {
		return ""
	}
	pods := make(map[string]*cluster.Pod, len(snap.Pods))
	for i := range snap.Pods {
		p := &snap.Pods[i]
		pods[p.Summary.Namespace+"/"+p.Summary.Name] = p
	}

	var b strings.Builder
	n := 0
	for _, e := range entities {
		parts := strings.Split(e.EntityKey, "/")
		if len(parts) != 3 || parts[1] != "pod" {
			continue
		}
		pod := pods[parts[0]+"/"+parts[2]]
		if pod == nil {
			continue
		}
		if n == maxPodStateEntries {
			break
		}
		n++
		fmt.Fprintf(&b, "%s:\n", e.EntityKey)
		for _, c := range pod.Containers {
			b.WriteString(describeContainer(c))
		}
	}
	return b.String()
}

// describeContainer 一行描述：当前状态 | 重启次数 | 上次退出（原因 @ 时间：信息）
func describeContainer(c cluster.PodContainerDetail) string {
	state := c.State
	if c.StateReason != "" {
		state += "(" + c.StateReason + ")"
	}
	line := fmt.Sprintf("  容器 %s: 当前 %s | 重启 %d 次", c.Name, state, c.RestartCount)
	if c.LastTerminationReason != "" {
		line += " | 上次退出: " + c.LastTerminationReason
		if c.LastTerminationTime != "" {
			line += " @ " + c.LastTerminationTime
		}
		if msg := strings.TrimSpace(c.LastTerminationMessage); msg != "" {
			if r := []rune(msg); len(r) > maxTerminationMsgChars {
				msg = string(r[:maxTerminationMsgChars]) + "…"
			}
			line += "（" + msg + "）"
		}
	}
	return line + "\n"
}
