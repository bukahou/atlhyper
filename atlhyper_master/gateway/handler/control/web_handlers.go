// control/web_handlers.go
// 命令生成工具函数
package control

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// GenID 生成唯一命令ID（用于审计、ACK回执关联）。
// 说明：ID 只是"这次下发"的唯一性标识；真正避免重复执行靠 Idem（幂等键）。
func GenID() string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("cmd-%d", timeNowUnixNano())))
	return "cmd-" + hex.EncodeToString(sum[:8])
}

// timeNowUnixNano：可替换的时间函数（便于单元测试打桩）。
var timeNowUnixNano = func() int64 { return time.Now().UnixNano() }

// Idem 计算幂等键（避免重复执行）。
// 典型输入：动作 + 集群ID + 资源定位 + 关键参数（如 replicas / newImage）。
func Idem(parts ...any) string {
	h := sha256.New()
	for _, p := range parts {
		fmt.Fprint(h, "|", p)
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
