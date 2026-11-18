// atlhyper_master_v2/notifier/alert/severity.go
// 严重度四档及其比较
package alert

// Severity 严重度，与 AIOps 事件的四档一致
type Severity string

const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// severityRanks 档位从低到高的序号；不在表里的取值序号为 0
var severityRanks = map[Severity]int{
	SeverityLow:      1,
	SeverityMedium:   2,
	SeverityHigh:     3,
	SeverityCritical: 4,
}

// ParseSeverity 把字符串解析为严重度；只接受四档的小写写法
func ParseSeverity(text string) (Severity, bool) {
	severity := Severity(text)
	if _, known := severityRanks[severity]; !known {
		return "", false
	}
	return severity, true
}

// Rank 档位序号：low=1 … critical=4；认不出的取值为 0
func (s Severity) Rank() int { return severityRanks[s] }

// AtLeast 严重度是否达到门槛。
//
// 认不出的严重度不满足任何门槛（宁可不发，也不把来历不明的告警当高严重度）；
// 认不出的门槛按最低处理（配置坏了不应让告警全部发不出去）
func (s Severity) AtLeast(min Severity) bool {
	rank := s.Rank()
	return rank > 0 && rank >= min.Rank()
}

// MaxSeverity 两者中较高的一档
func MaxSeverity(a, b Severity) Severity {
	if b.Rank() > a.Rank() {
		return b
	}
	return a
}
