// atlhyper_master_v2/gateway/handler/auth_permissions.go
// 完整权限表（角色说明页用；权限表本身不含任何集群数据，0 级可见）
package handler

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/gateway/middleware"
)

// PermissionTableRow 一条权限：名称 + 授予它的最低等级（该等级及以上都拥有）
type PermissionTableRow struct {
	Permission string `json:"permission"`
	MinLevel   int    `json:"minLevel"`
}

// PermissionTableResponse 权限表：levels 为全部角色等级（从低到高），rows 按授予等级排序
type PermissionTableResponse struct {
	Levels []int                `json:"levels"`
	Rows   []PermissionTableRow `json:"rows"`
}

// GetPermissionTable 返回完整权限表
// GET /api/v2/auth/permissions
func GetPermissionTable(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	table := middleware.PermissionTable()
	resp := PermissionTableResponse{
		Levels: []int{middleware.RoleMember, middleware.RoleViewer, middleware.RoleOperator, middleware.RoleAdmin},
		Rows:   make([]PermissionTableRow, len(table)),
	}
	for i, g := range table {
		resp.Rows[i] = PermissionTableRow{Permission: string(g.Permission), MinLevel: g.MinLevel}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    resp,
	})
}
