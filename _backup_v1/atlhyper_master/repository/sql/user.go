// repository/sql/user.go
//
// 用户仓库 SQL 实现
//
// 本文件实现了 repository.UserRepository 接口，提供用户数据的 CRUD 操作。
// 这是一个纯业务逻辑层，所有 SQL 语句通过全局变量 Q 访问。
//
// 实现的接口方法:
//   - GetByID: 根据ID查询用户
//   - GetByUsername: 根据用户名查询用户
//   - GetAll: 获取所有用户列表
//   - ExistsByUsername: 检查用户名是否存在
//   - ExistsByID: 检查用户ID是否存在
//   - Count: 统计用户总数
//   - Insert: 创建新用户
//   - UpdateRole: 更新用户角色
//   - UpdateLastLogin: 更新最后登录时间
//   - Delete: 删除用户
//
// 数据库表: users
// 字段: id, username, password_hash, display_name, email, role, created_at, last_login
package sql

import (
	"context"
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master/model/entity"
	"AtlHyper/atlhyper_master/store"
)

// ============================================================================
// UserRepo 用户仓库实现
// ============================================================================

// UserRepo 用户仓库的 SQL 实现
// 实现 repository.UserRepository 接口
// 所有方法都使用 store.DB 执行数据库操作
type UserRepo struct{}

// ============================================================================
// 查询方法
// ============================================================================

// GetByID 根据用户ID查询用户信息
//
// 参数:
//   - ctx: 上下文，用于超时控制和取消
//   - id: 用户ID
//
// 返回:
//   - *entity.User: 用户实体，包含完整用户信息
//   - error: 查询失败或用户不存在时返回错误 (sql.ErrNoRows)
//
// 使用场景:
//   - 获取当前登录用户信息
//   - 用户详情页展示
//   - 权限校验时获取用户角色
func (r *UserRepo) GetByID(ctx context.Context, id int) (*entity.User, error) {
	row := store.DB.QueryRowContext(ctx, Q.User.GetByID, id)
	return scanUser(row)
}

// GetByUsername 根据用户名查询用户信息
//
// 参数:
//   - ctx: 上下文
//   - username: 用户名 (唯一标识)
//
// 返回:
//   - *entity.User: 用户实体
//   - error: 查询失败或用户不存在时返回错误
//
// 使用场景:
//   - 用户登录验证 (获取密码哈希进行比对)
//   - 检查用户名是否已被使用
func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*entity.User, error) {
	row := store.DB.QueryRowContext(ctx, Q.User.GetByUsername, username)
	return scanUser(row)
}

// GetAll 获取所有用户列表
//
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - []entity.User: 用户列表，按ID升序排列
//   - error: 查询失败时返回错误
//
// 使用场景:
//   - 管理界面用户列表展示
//   - 用户导出功能
//
// 注意:
//   - 返回所有字段包括 password_hash，调用方应注意脱敏
func (r *UserRepo) GetAll(ctx context.Context) ([]entity.User, error) {
	rows, err := store.DB.QueryContext(ctx, Q.User.GetAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []entity.User
	for rows.Next() {
		u, err := scanUserFromRows(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, *u)
	}
	return users, rows.Err()
}

// ============================================================================
// 存在性检查方法
// ============================================================================

// ExistsByUsername 检查用户名是否已存在
//
// 参数:
//   - ctx: 上下文
//   - username: 待检查的用户名
//
// 返回:
//   - bool: true 表示用户名已存在
//   - error: 查询失败时返回错误
//
// 使用场景:
//   - 用户注册时验证用户名唯一性
//   - 用户名修改前的预检查
func (r *UserRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int
	err := store.DB.QueryRowContext(ctx, Q.User.ExistsByUsername, username).Scan(&count)
	return count > 0, err
}

// ExistsByID 检查用户ID是否存在
//
// 参数:
//   - ctx: 上下文
//   - id: 待检查的用户ID
//
// 返回:
//   - bool: true 表示用户存在
//   - error: 查询失败时返回错误
//
// 使用场景:
//   - 操作前验证用户是否有效
//   - 关联数据的外键检查
func (r *UserRepo) ExistsByID(ctx context.Context, id int) (bool, error) {
	var count int
	err := store.DB.QueryRowContext(ctx, Q.User.ExistsByID, id).Scan(&count)
	return count > 0, err
}

// Count 统计用户总数
//
// 参数:
//   - ctx: 上下文
//
// 返回:
//   - int: 用户总数
//   - error: 查询失败时返回错误
//
// 使用场景:
//   - 判断是否需要初始化管理员账户 (count == 0)
//   - 用户管理页面的统计信息
func (r *UserRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := store.DB.QueryRowContext(ctx, Q.User.Count).Scan(&count)
	return count, err
}

// ============================================================================
// 写入方法
// ============================================================================

// Insert 创建新用户
//
// 参数:
//   - ctx: 上下文
//   - u: 用户实体，需要填充以下字段:
//   - Username: 用户名 (必填，唯一)
//   - PasswordHash: 密码哈希 (必填，应使用 bcrypt)
//   - DisplayName: 显示名称
//   - Email: 邮箱地址
//   - Role: 角色 (1=管理员, 2=普通用户)
//   - CreatedAt: 会被覆盖为当前UTC时间
//
// 返回:
//   - int64: 新创建用户的ID
//   - error: 创建失败时返回错误 (如用户名重复)
//
// 注意:
//   - 密码应在调用前使用 bcrypt 哈希
//   - SQLite 使用 LastInsertId() 获取ID
//   - PostgreSQL 使用 RETURNING id (需要不同的调用方式)
func (r *UserRepo) Insert(ctx context.Context, u *entity.User) (int64, error) {
	res, err := store.DB.ExecContext(ctx, Q.User.Insert,
		u.Username, u.PasswordHash, u.DisplayName, u.Email, u.Role,
		time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateRole 更新用户角色
//
// 参数:
//   - ctx: 上下文
//   - id: 用户ID
//   - role: 新角色值 (1=管理员, 2=普通用户)
//
// 返回:
//   - error: 更新失败时返回错误
//
// 使用场景:
//   - 管理员提升/降级用户权限
func (r *UserRepo) UpdateRole(ctx context.Context, id int, role int) error {
	_, err := store.DB.ExecContext(ctx, Q.User.UpdateRole, role, id)
	return err
}

// UpdateLastLogin 更新用户最后登录时间
//
// 参数:
//   - ctx: 上下文
//   - id: 用户ID
//
// 返回:
//   - error: 更新失败时返回错误
//
// 使用场景:
//   - 用户成功登录后调用
//   - 记录用户活跃度
//
// 时间格式: RFC3339 (2006-01-02T15:04:05Z07:00)
func (r *UserRepo) UpdateLastLogin(ctx context.Context, id int) error {
	_, err := store.DB.ExecContext(ctx, Q.User.UpdateLastLogin,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// Delete 删除用户
//
// 参数:
//   - ctx: 上下文
//   - id: 待删除的用户ID
//
// 返回:
//   - error: 删除失败时返回错误
//
// 注意:
//   - 这是物理删除，数据不可恢复
//   - 建议在删除前检查是否为最后一个管理员
//   - 关联的审计日志不会被删除
func (r *UserRepo) Delete(ctx context.Context, id int) error {
	_, err := store.DB.ExecContext(ctx, Q.User.Delete, id)
	return err
}

// ============================================================================
// 内部辅助函数
// ============================================================================

// scanUser 从单行查询结果扫描用户数据
//
// 参数:
//   - row: sql.Row 单行查询结果
//
// 返回:
//   - *entity.User: 用户实体
//   - error: 扫描失败时返回错误 (包括 sql.ErrNoRows)
//
// 处理逻辑:
//   - created_at 和 last_login 存储为 TEXT (RFC3339 格式)
//   - last_login 可能为 NULL，使用 sql.NullString 处理
//   - 时间解析失败时使用零值
func scanUser(row *sql.Row) (*entity.User, error) {
	var u entity.User
	var createdAt, lastLogin sql.NullString

	// 扫描所有字段
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName,
		&u.Email, &u.Role, &createdAt, &lastLogin)
	if err != nil {
		return nil, err
	}

	// 解析 created_at 时间
	if createdAt.Valid {
		t, _ := time.Parse(time.RFC3339, createdAt.String)
		u.CreatedAt = t
	}

	// 解析 last_login 时间 (可选字段)
	if lastLogin.Valid {
		t, _ := time.Parse(time.RFC3339, lastLogin.String)
		u.LastLogin = &t
	}
	return &u, nil
}

// scanUserFromRows 从多行查询结果中扫描单个用户数据
//
// 参数:
//   - rows: sql.Rows 多行查询结果的当前行
//
// 返回:
//   - *entity.User: 用户实体
//   - error: 扫描失败时返回错误
//
// 注意:
//   - 与 scanUser 逻辑相同，但接收 sql.Rows 而非 sql.Row
//   - 用于 GetAll 等返回多条记录的查询
func scanUserFromRows(rows *sql.Rows) (*entity.User, error) {
	var u entity.User
	var createdAt, lastLogin sql.NullString

	err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.DisplayName,
		&u.Email, &u.Role, &createdAt, &lastLogin)
	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		t, _ := time.Parse(time.RFC3339, createdAt.String)
		u.CreatedAt = t
	}
	if lastLogin.Valid {
		t, _ := time.Parse(time.RFC3339, lastLogin.String)
		u.LastLogin = &t
	}
	return &u, nil
}
