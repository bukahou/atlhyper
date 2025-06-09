package rootcause

var InternalRootCauseDB = []RootCause{
	{
		Code:      "NodeNotReady",
		Source:    "Internal",
		Scope:     "Node",
		RootCause: "节点状态 NotReady，可能因资源压力或网络断联导致",
		MatchRules: []string{
			"NodeNotReady",
			"node.kubernetes.io/",
			"Node is not ready",
			"unreachable",
		},
		Priority: PriorityCritical,
	},
	{
		Code:      "OOMKilled",
		Source:    "Internal",
		Scope:     "Pod",
		RootCause: "容器因内存溢出被系统杀死（OOMKilled）",
		MatchRules: []string{
			"OOMKilled",
		},
		Priority: PriorityCritical,
	},
	{
		Code:      "CrashLoopBackOff",
		Source:    "Internal",
		Scope:     "Pod",
		RootCause: "容器启动后快速崩溃并被重复重启（CrashLoopBackOff）",
		MatchRules: []string{
			"CrashLoopBackOff",
			"BackOff",
		},
		Priority: PriorityCritical,
	},
	{
		Code:      "ProbeFailure",
		Source:    "Internal",
		Scope:     "Pod",
		RootCause: "探针检测失败，服务未启动或响应超时",
		MatchRules: []string{
			"Readiness probe failed",
			"Liveness probe failed",
			"Unhealthy",
			"probe error",
			"timeout awaiting probe",
		},
		Priority: PriorityCritical,
	},
	{
		Code:      "DeploymentProgressDeadlineExceeded",
		Source:    "Internal",
		Scope:     "Deployment",
		RootCause: "Deployment 更新超过最大允许时间，未完成滚动更新",
		MatchRules: []string{
			"ProgressDeadlineExceeded",
		},
		Priority: PriorityCritical,
	},
	{
		Code:      "NoReadyEndpoint",
		Source:    "Internal",
		Scope:     "Endpoint",
		RootCause: "所有 Pod 已从 Endpoints 剔除，服务无可用后端",
		MatchRules: []string{
			"NoReadyAddress",
		},
		Priority: PriorityCritical,
	},
	{
		Code:      "FailedCreatePodSandBox",
		Source:    "Internal",
		Scope:     "Pod",
		RootCause: "Pod 沙箱创建失败，容器运行时或 CNI 网络插件异常",
		MatchRules: []string{
			"FailedCreatePodSandBox",
		},
		Priority: PriorityCritical,
	},
}
