// service/user/service.go
// 用户服务 - 业务逻辑层
package user

import (
	"context"
	"errors"
	"fmt"

	"AtlHyper/atlhyper_master/config"
	"AtlHyper/atlhyper_master/gateway/middleware/auth"
	"AtlHyper/atlhyper_master/model/entity"
	"AtlHyper/atlhyper_master/repository"

	"golang.org/x/crypto/bcrypt"
)

// Login 用户登录
func Login(ctx context.Context, req LoginReq) (*LoginResult, error) {
	// 1. 查询用户
	u, err := repository.User.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	// 2. 校验密码
	if !checkPassword(req.Password, u.PasswordHash) {
		return nil, errors.New("密码错误")
	}

	// 3. 更新最后登录时间
	_ = repository.User.UpdateLastLogin(ctx, u.ID)

	// 4. 生成 Token
	token, err := auth.GenerateToken(u.ID, u.Username, u.Role)
	if err != nil {
		return nil, fmt.Errorf("生成 Token 失败: %w", err)
	}

	// 5. 获取集群列表
	clusterIDs, _ := repository.Mem.ListClusterIDs(ctx)

	// 6. 返回结果
	return &LoginResult{
		Token:      token,
		User:       toUserDTO(u),
		ClusterIDs: clusterIDs,
	}, nil
}

// Register 用户注册
func Register(ctx context.Context, req RegisterReq) (*UserDTO, error) {
	// 1. 校验密码长度
	minLen := config.GlobalConfig.JWT.MinPasswordLen
	if minLen == 0 {
		minLen = 6
	}
	if len(req.Password) < minLen {
		return nil, fmt.Errorf("密码长度不能少于 %d 位", minLen)
	}

	// 2. 检查用户名唯一性
	exists, err := repository.User.ExistsByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}
	if exists {
		return nil, errors.New("用户名已存在")
	}

	// 3. 加密密码
	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 4. 插入用户
	u := &entity.User{
		Username:     req.Username,
		PasswordHash: hashedPassword,
		DisplayName:  req.DisplayName,
		Email:        req.Email,
		Role:         req.Role,
	}

	id, err := repository.User.Insert(ctx, u)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	u.ID = int(id)
	dto := toUserDTO(u)
	return &dto, nil
}

// GetAllUsers 获取所有用户列表
func GetAllUsers(ctx context.Context) ([]UserDTO, error) {
	users, err := repository.User.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]UserDTO, len(users))
	for i, u := range users {
		dtos[i] = UserDTO{
			ID:          u.ID,
			Username:    u.Username,
			DisplayName: u.DisplayName,
			Email:       u.Email,
			Role:        u.Role,
			CreatedAt:   u.CreatedAt,
			LastLogin:   u.LastLogin,
		}
	}
	return dtos, nil
}

// UpdateRole 更新用户角色
func UpdateRole(ctx context.Context, req UpdateRoleReq) error {
	return repository.User.UpdateRole(ctx, req.ID, req.Role)
}

// Delete 删除用户
func Delete(ctx context.Context, userID, operatorID int) error {
	// 业务逻辑：不能删除自己
	if userID == operatorID {
		return errors.New("不能删除自己的账户")
	}

	// 检查用户是否存在
	exists, err := repository.User.ExistsByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if !exists {
		return errors.New("用户不存在")
	}

	return repository.User.Delete(ctx, userID)
}

// GetAuditLogs 获取用户审计日志
func GetAuditLogs(ctx context.Context) ([]AuditLogDTO, error) {
	logs, err := repository.Audit.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	dtos := make([]AuditLogDTO, len(logs))
	for i, log := range logs {
		dtos[i] = AuditLogDTO{
			ID:        log.ID,
			UserID:    log.UserID,
			Username:  log.Username,
			Role:      log.Role,
			Action:    log.Action,
			Success:   log.Success,
			IP:        log.IP,
			Method:    log.Method,
			Status:    log.Status,
			Timestamp: log.Timestamp,
		}
	}
	return dtos, nil
}

// ==================== 内部工具函数 ====================

// checkPassword 校验密码
func checkPassword(plain, hashed string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain)) == nil
}

// hashPassword 加密密码
func hashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// toUserDTO 转换为 DTO
func toUserDTO(u *entity.User) UserDTO {
	return UserDTO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		Email:       u.Email,
		Role:        u.Role,
		CreatedAt:   u.CreatedAt,
		LastLogin:   u.LastLogin,
	}
}
