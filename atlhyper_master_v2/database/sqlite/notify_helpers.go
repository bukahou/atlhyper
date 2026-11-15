// atlhyper_master_v2/database/sqlite/notify_helpers.go
// 通知各表共用的列编解码：时间一律按 UTC 的 RFC 3339 存（字符串比较即时间比较），集合按 JSON 存
package sqlite

import (
	"database/sql"
	"encoding/json"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
)

// 各通知方言的取得方法（方言本身没有状态）
func (d *Dialect) NotifyPolicy() database.NotifyPolicyDialect     { return &notifyPolicyDialect{} }
func (d *Dialect) NotifyAlert() database.NotifyAlertDialect       { return &notifyAlertDialect{} }
func (d *Dialect) NotifyDelivery() database.NotifyDeliveryDialect { return &notifyDeliveryDialect{} }
func (d *Dialect) NotifyReceiver() database.NotifyReceiverDialect { return &notifyReceiverDialect{} }
func (d *Dialect) NotifyRule() database.NotifyRuleDialect         { return &notifyRuleDialect{} }
func (d *Dialect) NotifySilence() database.NotifySilenceDialect   { return &notifySilenceDialect{} }

// utcStamp 时间 → 列值
func utcStamp(at time.Time) string { return at.UTC().Format(time.RFC3339) }

// nullableStamp 可空时间 → 列值；nil 写 NULL
func nullableStamp(at *time.Time) any {
	if at == nil {
		return nil
	}
	return utcStamp(*at)
}

// parseStamp 列值 → 时间；解析不了时返回零值
func parseStamp(text string) time.Time {
	at, _ := time.Parse(time.RFC3339, text)
	return at
}

// parseNullableStamp 可空列 → 时间指针
func parseNullableStamp(column sql.NullString) *time.Time {
	if !column.Valid || column.String == "" {
		return nil
	}
	at, err := time.Parse(time.RFC3339, column.String)
	if err != nil {
		return nil
	}
	return &at
}

// nullableInt 可空整数 → 列值
func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

// parseNullableInt 可空列 → 整数指针
func parseNullableInt(column sql.NullInt64) *int {
	if !column.Valid {
		return nil
	}
	value := int(column.Int64)
	return &value
}

// encodeJSON 集合 → 列值。这里的入参只有字符串 / 整数的切片与 map，编码不会失败
func encodeJSON(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "null"
	}
	return string(encoded)
}

// decodeStringMap 列值 → map；列值损坏时返回空 map，不让一行坏数据拖垮整个列表
func decodeStringMap(text string) map[string]string {
	decoded := map[string]string{}
	_ = json.Unmarshal([]byte(text), &decoded)
	if decoded == nil {
		return map[string]string{}
	}
	return decoded
}

// decodeStrings 列值 → 字符串切片（恒非 nil）
func decodeStrings(text string) []string {
	decoded := []string{}
	_ = json.Unmarshal([]byte(text), &decoded)
	if decoded == nil {
		return []string{}
	}
	return decoded
}

// decodeInt64s 列值 → 整数切片（恒非 nil）
func decodeInt64s(text string) []int64 {
	decoded := []int64{}
	_ = json.Unmarshal([]byte(text), &decoded)
	if decoded == nil {
		return []int64{}
	}
	return decoded
}

// pageClause 分页子句；limit ≤ 0 表示不分页
func pageClause(limit, offset int) (string, []any) {
	if limit <= 0 {
		return "", nil
	}
	if offset < 0 {
		offset = 0
	}
	return " LIMIT ? OFFSET ?", []any{limit, offset}
}
