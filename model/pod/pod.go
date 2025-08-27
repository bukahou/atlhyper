package pod

import "time"

// ====================== 顶层：Pod（直接发送这个结构体） ======================

type Pod struct {
	Summary    PodSummary       `json:"summary"`              // 概要（列表常用字段）
	Spec       PodSpec          `json:"spec"`                 // 关键调度/策略
	Containers []Container      `json:"containers"`           // 业务容器（必要）
	// 如需展示初始化/排障容器，可解开下面两行：
	// InitContainers      []Container `json:"initContainers,omitempty"`
	// EphemeralContainers []Container `json:"ephemeralContainers,omitempty"`
	Volumes   []Volume        `json:"volumes,omitempty"`      // 卷/存储
	Network   Network         `json:"network"`                // 网络信息
	Metrics   *PodMetrics     `json:"metrics,omitempty"`      // 运行时指标（可为空）
}

// ====================== summary ======================

type PodSummary struct {
	Name         string   `json:"name"`                      // Pod 名
	Namespace    string   `json:"namespace"`                 // 命名空间
	ControlledBy *Owner   `json:"controlledBy,omitempty"`    // 上层控制器（类型/名称）
	Phase        string   `json:"phase"`                     // 阶段：Pending/Running/...
	Ready        string   `json:"ready"`                     // 就绪/总数（如 "2/2"）
	Restarts     int32    `json:"restarts"`                  // 重启次数（容器总和）
	StartTime    time.Time`json:"startTime"`                 // 启动时间
	Age          string   `json:"age"`                       // 运行时长（派生显示）
	Node         string   `json:"node"`                      // 调度节点
	PodIP        string   `json:"podIP,omitempty"`           // Pod IP
	QoSClass     string   `json:"qosClass,omitempty"`        // Guaranteed/Burstable/BestEffort
	Reason       string   `json:"reason,omitempty"`          // 状态原因（非 Running 时）
	Message      string   `json:"message,omitempty"`         // 状态说明
	Badges       []string `json:"badges,omitempty"`          // UI 徽标（CrashLoop 等）
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
	Tolerations                   any               `json:"tolerations,omitempty"`            // 可直接透传 corev1.Toleration[]
	Affinity                      any               `json:"affinity,omitempty"`               // 可透传 *corev1.Affinity
	TopologySpreadConstraints     any               `json:"topologySpreadConstraints,omitempty"`
	RuntimeClassName              string            `json:"runtimeClassName,omitempty"`
	TerminationGracePeriodSeconds *int64            `json:"terminationGracePeriodSeconds,omitempty"`
}

// ====================== containers ======================

type Container struct {
	Name            string         `json:"name"`                             // 容器名
	Image           string         `json:"image"`                            // 镜像（建议含 digest）
	ImagePullPolicy string         `json:"imagePullPolicy"`                  // IfNotPresent/Always/Never
	Ports           []ContainerPort`json:"ports,omitempty"`                  // 端口
	Env             []EnvKV        `json:"env,omitempty"`                    // 环境变量（UI 友好简化）
	VolumeMounts    []VolumeMount  `json:"volumeMounts,omitempty"`           // 挂载
	Resources       Resources      `json:"resources,omitempty"`              // requests/limits
	Probes          *Probes        `json:"probes,omitempty"`                 // 探针
	SecurityContext any            `json:"securityContext,omitempty"`        // 可透传 *corev1.SecurityContext
	Status          ContainerStatus`json:"status"`                           // 状态摘要
}

type ContainerPort struct {
	Name          string `json:"name,omitempty"`
	ContainerPort int32  `json:"containerPort"`
	Protocol      string `json:"protocol"` // TCP/UDP/SCTP
}

type EnvKV struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"` // 若来自 CM/Secret，可在前端标注来源
}

type VolumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	ReadOnly  bool   `json:"readOnly,omitempty"`
	SubPath   string `json:"subPath,omitempty"`
}

type Resources struct {
	Requests map[string]string `json:"requests,omitempty"` // {"cpu":"200m","memory":"256Mi"}
	Limits   map[string]string `json:"limits,omitempty"`   // {"cpu":"1","memory":"512Mi"}
}

type Probes struct {
	Readiness any `json:"readiness,omitempty"` // 可透传 *corev1.Probe
	Liveness  any `json:"liveness,omitempty"`
	Startup   any `json:"startup,omitempty"`
}

type ContainerStatus struct {
	State                string `json:"state"`                          // Running/Waiting/Terminated
	RestartCount         int32  `json:"restartCount"`                   // 本容器重启次数
	LastTerminatedReason string `json:"lastTerminatedReason,omitempty"` // OOMKilled 等
}

// ====================== volumes ======================

type Volume struct {
	Name   string      `json:"name"`   // 卷名
	Type   string      `json:"type"`   // configMap/secret/pvc/emptyDir/hostPath/csi...
	Source interface{} `json:"source"` // 关键字段（如 configMap.name / pvc.claimName）
}

// ====================== network ======================

type Network struct {
	PodIP              string   `json:"podIP,omitempty"`  // 主 IP
	PodIPs             []string `json:"podIPs,omitempty"` // 双栈/多个 IP
	HostIP             string   `json:"hostIP,omitempty"` // 宿主机 IP
	HostNetwork        bool     `json:"hostNetwork"`      // 是否使用宿主网络
	DNSPolicy          string   `json:"dnsPolicy,omitempty"`
	ServiceAccountName string   `json:"serviceAccountName,omitempty"`
}

// ====================== metrics ======================

type PodMetrics struct {
	CPU    ResourceMetric `json:"cpu"`
	Memory ResourceMetric `json:"memory"`
}

type ResourceMetric struct {
	Usage   string  `json:"usage"`             // 如 "150m"/"220Mi"
	Limit   string  `json:"limit,omitempty"`   // 如 "1"/"512Mi"
	UtilPct float64 `json:"utilPct,omitempty"` // 0-100
}
