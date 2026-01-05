// model/k8s/pod.go
// Pod 资源模型
package k8s

import "time"

// ====================== 顶层：Pod ======================

type Pod struct {
	Summary    PodSummary   `json:"summary"`              // 概要（列表常用字段）
	Spec       PodSpec      `json:"spec"`                 // 关键调度/策略
	Containers []Container  `json:"containers"`           // 业务容器（必要）
	Volumes    []Volume     `json:"volumes,omitempty"`    // 卷/存储
	Network    Network      `json:"network"`              // 网络信息
	Metrics    *PodMetrics  `json:"metrics,omitempty"`    // 运行时指标（可为空）
}

// ====================== summary ======================

type PodSummary struct {
	Name         string    `json:"name"`                   // Pod 名
	Namespace    string    `json:"namespace"`              // 命名空间
	ControlledBy *Owner    `json:"controlledBy,omitempty"` // 上层控制器（类型/名称）
	Phase        string    `json:"phase"`                  // 阶段：Pending/Running/...
	Ready        string    `json:"ready"`                  // 就绪/总数（如 "2/2"）
	Restarts     int32     `json:"restarts"`               // 重启次数（容器总和）
	StartTime    time.Time `json:"startTime"`              // 启动时间
	Age          string    `json:"age"`                    // 运行时长（派生显示）
	Node         string    `json:"node"`                   // 调度节点
	PodIP        string    `json:"podIP,omitempty"`        // Pod IP
	QoSClass     string    `json:"qosClass,omitempty"`     // Guaranteed/Burstable/BestEffort
	Reason       string    `json:"reason,omitempty"`       // 状态原因（非 Running 时）
	Message      string    `json:"message,omitempty"`      // 状态说明
	Badges       []string  `json:"badges,omitempty"`       // UI 徽标（CrashLoop 等）
}

type Owner struct {
	Kind string `json:"kind"` // Deployment/StatefulSet/Job/DaemonSet...
	Name string `json:"name"` // 控制器名称
}

// ====================== spec ======================

type PodSpec struct {
	RestartPolicy                 string            `json:"restartPolicy"`
	PriorityClassName             string            `json:"priorityClassName,omitempty"`
	NodeSelector                  map[string]string `json:"nodeSelector,omitempty"`
	Tolerations                   any               `json:"tolerations,omitempty"`
	Affinity                      any               `json:"affinity,omitempty"`
	TopologySpreadConstraints     any               `json:"topologySpreadConstraints,omitempty"`
	RuntimeClassName              string            `json:"runtimeClassName,omitempty"`
	TerminationGracePeriodSeconds *int64            `json:"terminationGracePeriodSeconds,omitempty"`
}

// ====================== containers ======================

type Container struct {
	Name            string          `json:"name"`
	Image           string          `json:"image"`
	ImagePullPolicy string          `json:"imagePullPolicy"`
	Ports           []ContainerPort `json:"ports,omitempty"`
	Env             []EnvKV         `json:"env,omitempty"`
	VolumeMounts    []VolumeMount   `json:"volumeMounts,omitempty"`
	Resources       Resources       `json:"resources,omitempty"`
	Probes          *Probes         `json:"probes,omitempty"`
	SecurityContext any             `json:"securityContext,omitempty"`
	Status          ContainerStatus `json:"status"`
}

type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"`
}

type EnvKV struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
	SubPath   string `json:"subPath,omitempty"`
}

type Resources struct {
	Requests map[string]string `json:"requests,omitempty"`
	Limits   map[string]string `json:"limits,omitempty"`
}

type Probes struct {
	Readiness any `json:"readiness,omitempty"`
	Liveness  any `json:"liveness,omitempty"`
	Startup   any `json:"startup,omitempty"`
}

type ContainerStatus struct {
	State                string `json:"state"`
	RestartCount         int32  `json:"restartCount"`
	LastTerminatedReason string `json:"lastTerminatedReason,omitempty"`
}

// ====================== volumes ======================

type Volume struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	Source interface{} `json:"source"`
}

// ====================== network ======================

type Network struct {
	PodIP              string   `json:"podIP,omitempty"`
	PodIPs             []string `json:"podIPs,omitempty"`
	HostIP             string   `json:"hostIP,omitempty"`
	HostNetwork        bool     `json:"hostNetwork"`
	DNSPolicy          string   `json:"dnsPolicy,omitempty"`
	ServiceAccountName string   `json:"serviceAccountName,omitempty"`
}

// ====================== metrics ======================

type PodMetrics struct {
	CPU    ResourceMetric `json:"cpu"`
	Memory ResourceMetric `json:"memory"`
}

type ResourceMetric struct {
	Usage   string  `json:"usage"`
	Limit   string  `json:"limit,omitempty"`
	UtilPct float64 `json:"utilPct,omitempty"`
}
