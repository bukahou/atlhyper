package user

import (
	"AtlHyper/atlhyper_master/config"
	"AtlHyper/atlhyper_master/db/utils"
	"AtlHyper/atlhyper_master/model"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// =======================================================================
// ✅ RegisterUser：注册新用户（插入一条用户记录）
//
// 功能说明：
//   - 校验用户名唯一性
//   - 对明文密码进行 bcrypt 加密
//   - 插入用户信息到 users 表
//
// 参数：
//   - username：用户名，唯一
//   - password：明文密码
//   - displayName：展示名称（可用于昵称）
//   - email：邮箱（可选）
//   - role：角色（例如 1 = 普通用户，3 = 管理员）
//
// 返回值：
//   - models.User：注册成功后的用户信息（不含密码）
//   - error：若插入失败或用户已存在，则返回错误
// =======================================================================
func RegisterUser(username, password, displayName, email string, role int) (*model.User, error) {
	// 0️⃣ 检查密码长度
	minLen := config.GlobalConfig.JWT.MinPasswordLen
	if minLen == 0 {
		minLen = 6 // 默认最小 6 位
	}
	if len(password) < minLen {
		return nil, fmt.Errorf("密码长度不能少于 %d 位", minLen)
	}

	// 1️⃣ 检查用户名是否已存在
	var exists int
	err := utils.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, username).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, errors.New("用户名已存在")
	}

	// 2️⃣ 加密密码（bcrypt）
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 3️⃣ 构造插入时间（ISO 8601 格式）
	now := time.Now().UTC().Format(time.RFC3339)

	// 4️⃣ 执行插入操作
	res, err := utils.DB.Exec(`
		INSERT INTO users (username, password_hash, display_name, email, role, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		username, string(hashed), displayName, email, role, now,
	)
	if err != nil {
		return nil, err
	}

	// 5️⃣ 获取插入后的用户 ID
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	// 6️⃣ 返回构造的用户信息（不包含密码）
	return &model.User{
		ID:           int(id),
		Username:     username,
		DisplayName:  displayName,
		Email:        email,
		Role:         role,
		CreatedAt:    time.Now().UTC(),
		LastLogin:    nil,
	}, nil
}


// =======================================================================
// ✅ UpdateUserRole：根据用户 ID 修改用户角色
//
// 参数：
//   - id：用户的唯一 ID
//   - newRole：新的角色值（例如 1=普通用户，3=管理员）
//
// 返回 error：执行成功返回 nil，否则返回错误信息
// =======================================================================
func UpdateUserRole(id int, newRole int) error {
	_, err := utils.DB.Exec(`UPDATE users SET role = ? WHERE id = ?`, newRole, id)
	if err != nil {
		return fmt.Errorf("修改用户角色失败: %w", err)
	}
	return nil
}


// =======================================================================
// ✅ DeleteUser：根据用户 ID 删除用户
//
// 功能说明：
//   - 检查用户是否存在
//   - 执行硬删除（从数据库中移除用户记录）
//
// 参数：
//   - id：要删除的用户 ID
//   - operatorID：执行删除操作的管理员 ID（用于防止自删）
//
// 返回值：
//   - error：若删除失败或用户不存在，则返回错误
// =======================================================================
func DeleteUser(id int, operatorID int) error {
	// 1️⃣ 检查是否删除自己
	if id == operatorID {
		return errors.New("不能删除自己的账户")
	}

	// 2️⃣ 检查用户是否存在
	var exists int
	err := utils.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE id = ?`, id).Scan(&exists)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if exists == 0 {
		return errors.New("用户不存在")
	}

	// 3️⃣ 执行删除操作
	_, err = utils.DB.Exec(`DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}

	return nil
}