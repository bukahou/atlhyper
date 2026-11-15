// atlhyper_master_v2/notifier/pipeline/link.go
// 拓扑归并的抑制：受影响事件的告警在它归到的源头告警持续时不发；源头的消息带上影响范围
package pipeline

import (
	"sort"

	"AtlHyper/atlhyper_master_v2/notifier/alert"
)

// applyLink 记下或去掉一条关联
func (p *AlertPipeline) applyLink(update linkUpdate) {
	if update.root == "" {
		delete(p.inhibitors, update.child)
		return
	}
	p.inhibitors[update.child] = update
}

// rootFiring 这条告警归到的源头告警是否正在持续
func (p *AlertPipeline) rootFiring(fingerprint string) bool {
	link, ok := p.inhibitors[fingerprint]
	if !ok {
		return false
	}
	root := p.alerts[link.root]
	return root != nil && root.alert.Status == alert.StatusFiring
}

// impactedSubjects 归到这条告警下面的受影响主体（排序，结果稳定）。受影响事件不一定有告警（只到警告），所以从关联里取
func (p *AlertPipeline) impactedSubjects(rootFingerprint string) []string {
	var subjects []string
	for _, link := range p.inhibitors {
		if link.root == rootFingerprint && link.subject != "" {
			subjects = append(subjects, link.subject)
		}
	}
	sort.Strings(subjects)
	return subjects
}
