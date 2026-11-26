// atlhyper_master_v2/model/service_account_token_validation.go
// 服务账号与访问令牌的输入校验（纯函数）
package model

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// ErrInvalidServiceAccountInput 新建、修改服务账号或签发令牌时的输入不合规；接口据此返回 400
var ErrInvalidServiceAccountInput = errors.New("服务账号信息不合规")

const (
	minServiceAccountNameLength = 3
	maxServiceAccountNameLength = 32
	// maxServiceAccountDisplayNameRunes / maxAPITokenNameRunes 字数上限（按字符计，不按字节）
	maxServiceAccountDisplayNameRunes = 64
	maxAPITokenNameRunes              = 64
)

// ValidateServiceAccountName 校验账号名：3–32 个字符，只允许小写字母、数字与连字符，以字母开头。
//
// 不做去空白之类的自动修正：账号名会出现在审计与日志里，改写过的名字与管理员输入的对不上。
func ValidateServiceAccountName(name string) error {
	if len(name) < minServiceAccountNameLength || len(name) > maxServiceAccountNameLength {
		return fmt.Errorf("%w：账号名的长度须为 %d–%d 个字符", ErrInvalidServiceAccountInput,
			minServiceAccountNameLength, maxServiceAccountNameLength)
	}
	for i := 0; i < len(name); i++ {
		char := name[i]
		isLetter := char >= 'a' && char <= 'z'
		if isLetter || (i > 0 && ((char >= '0' && char <= '9') || char == '-')) {
			continue
		}
		return fmt.Errorf("%w：账号名只能包含小写字母、数字与连字符，且以字母开头", ErrInvalidServiceAccountInput)
	}
	return nil
}

// ValidateServiceAccountProfile 校验显示名（可空，有字数上限，不得含控制字符）与等级（必填，0–ServiceAccountMaxRole）
func ValidateServiceAccountProfile(displayName string, role *int) error {
	if utf8.RuneCountInString(displayName) > maxServiceAccountDisplayNameRunes {
		return fmt.Errorf("%w：显示名不能超过 %d 个字", ErrInvalidServiceAccountInput, maxServiceAccountDisplayNameRunes)
	}
	if containsControlRune(displayName) {
		return fmt.Errorf("%w：显示名不能包含控制字符", ErrInvalidServiceAccountInput)
	}
	if role == nil {
		return fmt.Errorf("%w：缺少等级", ErrInvalidServiceAccountInput)
	}
	if *role < 0 || *role > ServiceAccountMaxRole {
		return fmt.Errorf("%w：服务账号的等级须为 0–%d", ErrInvalidServiceAccountInput, ServiceAccountMaxRole)
	}
	return nil
}

// ValidateUpdateServiceAccountRequest 修改是整体替换：显示名、等级、启停三项都必须给出
func ValidateUpdateServiceAccountRequest(req *UpdateServiceAccountRequest) error {
	if req.DisplayName == nil {
		return fmt.Errorf("%w：缺少显示名", ErrInvalidServiceAccountInput)
	}
	if req.Enabled == nil {
		return fmt.Errorf("%w：缺少启停状态", ErrInvalidServiceAccountInput)
	}
	return ValidateServiceAccountProfile(*req.DisplayName, req.Role)
}

// ValidateAPITokenRequest 校验令牌名称（1–64 个字，不得含控制字符）与有效期（不给 = 不过期；给了须为 1–上限天）
func ValidateAPITokenRequest(name string, expiresInDays *int) error {
	if name == "" {
		return fmt.Errorf("%w：令牌名称不能为空", ErrInvalidServiceAccountInput)
	}
	if utf8.RuneCountInString(name) > maxAPITokenNameRunes {
		return fmt.Errorf("%w：令牌名称不能超过 %d 个字", ErrInvalidServiceAccountInput, maxAPITokenNameRunes)
	}
	if containsControlRune(name) {
		return fmt.Errorf("%w：令牌名称不能包含控制字符", ErrInvalidServiceAccountInput)
	}
	if expiresInDays != nil && (*expiresInDays < 1 || *expiresInDays > MaxAPITokenExpiresInDays) {
		return fmt.Errorf("%w：有效期须为 1–%d 天", ErrInvalidServiceAccountInput, MaxAPITokenExpiresInDays)
	}
	return nil
}
