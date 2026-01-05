// service/event/overview.go
package event

import (
	"strings"
	"time"

	"AtlHyper/atlhyper_master/db/repository/eventlog"
	"AtlHyper/atlhyper_master/model/ui"
	model "AtlHyper/model/transport"
)

// getJST returns a non-nil *time.Location.
// It tries "Asia/Tokyo"; if tzdata is missing, it falls back to a fixed +09:00 zone.
func getJST() *time.Location {
	// Prefer the OS/env-provided local zone if available
	if time.Local != nil {
		return time.Local
	}
	if loc, err := time.LoadLocation("Asia/Tokyo"); err == nil && loc != nil {
		return loc
	}
	return time.FixedZone("JST", 9*3600)
}

func BuildEventOverview(clusterID string, withinDays int) (*ui.EventOverviewDTO, error) {
	// 1) 计算 since：当前东京时间减 withinDays 天 → RFC3339
	loc := getJST()
	now := time.Now().In(loc)
	if withinDays < 0 {
		withinDays = 0
	}
	since := now.Add(-time.Duration(withinDays) * 24 * time.Hour).Format(time.RFC3339)

	// 2) 明细（把 RFC3339 传给底层）
	rows, err := eventlog.GetEventLogsSince(clusterID, since)
	if err != nil {
		return nil, err
	}
	// 若 rows 为 nil，保证 JSON 输出为 [] 而不是 null
	if rows == nil {
		rows = []model.EventLog{}
	}

	// 3) 聚合
	sevMap := map[string]int{}
	catSet := map[string]struct{}{}
	kindSet := map[string]struct{}{}
	eventCategoryCount := 0

	for _, e := range rows {
		sev := strings.ToLower(strings.TrimSpace(e.Severity))
		if sev == "" || sev == "normal" {
			sev = "info"
		}
		sevMap[sev]++

		cat := strings.TrimSpace(e.Category)
		if cat == "" {
			cat = "Unknown"
		}
		catSet[cat] = struct{}{}
		if strings.EqualFold(cat, "Event") {
			eventCategoryCount++
		}

		kind := strings.TrimSpace(e.Kind)
		if kind == "" {
			kind = "Unknown"
		}
		kindSet[kind] = struct{}{}
	}

	info := sevMap["info"]
	warn := sevMap["warning"]
	errc := sevMap["error"]

	cards := ui.EventCards{
		TotalAlerts:     len(rows),          // 数据总条数
		TotalEvents:     eventCategoryCount, // Category=Event 的条数
		Info:            info,
		Warning:         warn,
		Error:           errc,
		CategoriesCount: len(catSet),
		KindsCount:      len(kindSet),
	}

	return &ui.EventOverviewDTO{Cards: cards, Rows: rows}, nil
}
