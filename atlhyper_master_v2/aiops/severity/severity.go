// Package severity AIOps 事件严重度的取值与先后。
// 只此一份：状态机升档、后台分析的触发门槛、库里「只升不降」的更新语句都用它。
package severity

import (
	"strconv"
	"strings"
)

const (
	Low      = "low"
	Medium   = "medium"
	High     = "high"
	Critical = "critical"
)

// ascending 从轻到重
var ascending = []string{Low, Medium, High, Critical}

// Rank 档位，越大越严重；不认识的取值为 0
func Rank(s string) int {
	for i, v := range ascending {
		if v == s {
			return i + 1
		}
	}
	return 0
}

// AtLeast s 是否达到 min（min 不认识时一律达到）
func AtLeast(s, min string) bool {
	return Rank(s) >= Rank(min)
}

// RankSQL 把 SQL 表达式（列名或占位符）映射为与 Rank 相同的档位，供「只升不降」的更新在库里比较
func RankSQL(expr string) string {
	var b strings.Builder
	b.WriteString("CASE ")
	b.WriteString(expr)
	for i := len(ascending) - 1; i >= 0; i-- {
		b.WriteString(" WHEN '")
		b.WriteString(ascending[i])
		b.WriteString("' THEN ")
		b.WriteString(strconv.Itoa(i + 1))
	}
	b.WriteString(" ELSE 0 END")
	return b.String()
}
