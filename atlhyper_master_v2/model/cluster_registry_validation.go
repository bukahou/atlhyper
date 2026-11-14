// atlhyper_master_v2/model/cluster_registry_validation.go
// 集群注册表的输入校验（纯函数）
package model

import (
	"errors"
	"fmt"
	"unicode"
	"unicode/utf8"
)

// ErrInvalidClusterInput 登记或修改集群时的输入不合规；接口据此返回 400
var ErrInvalidClusterInput = errors.New("集群信息不合规")

const (
	// maxClusterIDLength 集群 ID 的长度上限。
	// 与 DNS 标签同长：集群 ID 会出现在 URL 路径、请求头与日志里，短而规整的 ID 不需要转义
	maxClusterIDLength = 63
	// maxClusterNameRunes / maxClusterDescriptionRunes 显示名与描述的字数上限（按字符计，不按字节）
	maxClusterNameRunes        = 64
	maxClusterDescriptionRunes = 256
)

// ValidateClusterID 校验集群 ID：1–63 个字符，只允许小写字母、数字、- 和 .，首尾必须是字母或数字。
//
// 不做去空白之类的自动修正：ID 要与 Agent 上报的逐字一致，改写后的 ID 永远对不上。
func ValidateClusterID(clusterID string) error {
	if len(clusterID) == 0 || len(clusterID) > maxClusterIDLength {
		return fmt.Errorf("%w：集群 ID 的长度须为 1–%d 个字符", ErrInvalidClusterInput, maxClusterIDLength)
	}
	for i := 0; i < len(clusterID); i++ {
		char := clusterID[i]
		if isLowerAlphanumeric(char) {
			continue
		}
		edge := i == 0 || i == len(clusterID)-1
		if (char == '-' || char == '.') && !edge {
			continue
		}
		return fmt.Errorf("%w：集群 ID 只能包含小写字母、数字、- 和 .，且首尾必须是字母或数字", ErrInvalidClusterInput)
	}
	return nil
}

func isLowerAlphanumeric(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
}

// ValidateClusterProfile 校验显示名与描述：显示名必填；两者都有字数上限，且不得含控制字符
func ValidateClusterProfile(name, description string) error {
	if name == "" {
		return fmt.Errorf("%w：显示名不能为空", ErrInvalidClusterInput)
	}
	if utf8.RuneCountInString(name) > maxClusterNameRunes {
		return fmt.Errorf("%w：显示名不能超过 %d 个字", ErrInvalidClusterInput, maxClusterNameRunes)
	}
	if utf8.RuneCountInString(description) > maxClusterDescriptionRunes {
		return fmt.Errorf("%w：描述不能超过 %d 个字", ErrInvalidClusterInput, maxClusterDescriptionRunes)
	}
	if containsControlRune(name) || containsControlRune(description) {
		return fmt.Errorf("%w：显示名与描述不能包含控制字符", ErrInvalidClusterInput)
	}
	return nil
}

func containsControlRune(text string) bool {
	for _, char := range text {
		if unicode.IsControl(char) {
			return true
		}
	}
	return false
}
