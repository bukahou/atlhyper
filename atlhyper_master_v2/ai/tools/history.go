package tools

import (
	"context"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"

	"AtlHyper/atlhyper_master_v2/ai/toolkit"
	"AtlHyper/atlhyper_master_v2/database"
)

const maxDeployHistory = 20

type deployHistoryParams struct {
	Path  string `json:"path" jsonschema:"部署配置里登记的 kustomize 路径"`
	Limit int    `json:"limit,omitempty" jsonschema:"返回最近 N 条（不填为 5）"`
}

func deployHistoryTool(repo database.DeployHistoryRepository) *toolkit.Tool {
	return toolkit.MustDefine(toolkit.Spec[deployHistoryParams]{
		Name:        "get_deploy_history",
		Description: "查询某个部署路径最近的部署记录：提交号、部署状态、触发方式。排查「是不是最近的变更引起的」时用。",
		Permission:  toolkit.PermDeployHistory,
		Refine: func(s *jsonschema.Schema) {
			s.Properties["limit"].Minimum = ptr(1.0)
			s.Properties["limit"].Maximum = ptr(float64(maxDeployHistory))
		},
		Handler: func(ctx context.Context, clusterID string, p deployHistoryParams) (string, error) {
			records, err := repo.List(ctx, database.DeployHistoryQueryOpts{
				ClusterID: clusterID, Path: p.Path, Limit: orDefault(p.Limit, 5),
			})
			if err != nil {
				return "", fmt.Errorf("查询部署历史失败: %w", err)
			}
			if len(records) == 0 {
				return fmt.Sprintf("路径 '%s' 没有部署记录", p.Path), nil
			}
			return marshal(map[string]any{"path": p.Path, "records": records, "count": len(records)})
		},
	})
}
