package config

import (
	"AtlHyper/atlhyper_master/db/utils"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// 全量返回用（给 WebUI 展示也方便）
type SlackConfigRow struct {
	ID          int64
	Name        string
	Enable      int    // 0/1
	Webhook     string // 可能为空
	IntervalSec int64  // 秒
	UpdatedAt   time.Time
}

// GetSlackConfigFull 返回 config 表中 name='slack' 的完整一行；
// 若无记录，返回默认值（Enable=0, Webhook="", IntervalSec=5, Name="slack"）
func GetSlackConfigFull() (SlackConfigRow, error) {
	var (
		id  sql.NullInt64
		nm  sql.NullString
		en  sql.NullInt64
		wh  sql.NullString
		iv  sql.NullInt64
		upd sql.NullString
	)
	err := utils.DB.QueryRow(`
		SELECT id, name, enable, webhook, interval_sec, updated_at
		FROM config
		WHERE name = 'slack'
	`).Scan(&id, &nm, &en, &wh, &iv, &upd)

	// 查无记录：给一行默认值
	if err == sql.ErrNoRows {
		return SlackConfigRow{
			ID:          0,
			Name:        "slack",
			Enable:      0,
			Webhook:     "",
			IntervalSec: 5,
			UpdatedAt:   time.Time{},
		}, nil
	}
	if err != nil {
		// 出错也回默认值，但把错误返回
		return SlackConfigRow{
			ID:          0,
			Name:        "slack",
			Enable:      0,
			Webhook:     "",
			IntervalSec: 5,
			UpdatedAt:   time.Time{},
		}, err
	}

	row := SlackConfigRow{
		ID:          0,
		Name:        "slack",
		Enable:      0,
		Webhook:     "",
		IntervalSec: 5,
		UpdatedAt:   time.Time{},
	}
	if id.Valid {
		row.ID = id.Int64
	}
	if nm.Valid && strings.TrimSpace(nm.String) != "" {
		row.Name = strings.TrimSpace(nm.String)
	}
	if en.Valid && en.Int64 != 0 {
		row.Enable = 1
	}
	if wh.Valid {
		row.Webhook = strings.TrimSpace(wh.String)
	}
	if iv.Valid && iv.Int64 > 0 {
		row.IntervalSec = iv.Int64
	}
	if upd.Valid {
		if t, e := time.Parse(time.RFC3339, upd.String); e == nil {
			row.UpdatedAt = t
		}
	}
	return row, nil
}

// 更新请求（指针表示“可选字段”：nil 则不更新）
type SlackConfigUpdate struct {
	Enable      *int
	Webhook     *string
	IntervalSec *int64
}

// UpdateSlackConfig 按需更新 id=1 的 slack 配置（部分字段更新）
// - 如果所有字段都为 nil，则不执行更新
// - 会自动更新 updated_at
func UpdateSlackConfig(upd SlackConfigUpdate) error {
	setParts := make([]string, 0, 3)
	args := make([]any, 0, 4)

	if upd.Enable != nil {
		en := 0
		if *upd.Enable != 0 {
			en = 1
		}
		setParts = append(setParts, "enable=?")
		args = append(args, en)
	}
	if upd.Webhook != nil {
		w := strings.TrimSpace(*upd.Webhook)
		setParts = append(setParts, "webhook=?")
		args = append(args, w)
	}
	if upd.IntervalSec != nil {
		sec := *upd.IntervalSec
		if sec <= 0 {
			sec = 5 // 兜底
		}
		setParts = append(setParts, "interval_sec=?")
		args = append(args, sec)
	}

	if len(setParts) == 0 {
		// 没有要更新的字段，直接返回
		return nil
	}

	// 加上 updated_at
	setParts = append(setParts, "updated_at=?")
	args = append(args, time.Now().Format(time.RFC3339))

	query := fmt.Sprintf(`UPDATE config SET %s WHERE id=1`, strings.Join(setParts, ", "))
	_, err := utils.DB.Exec(query, args...)
	return err
}
