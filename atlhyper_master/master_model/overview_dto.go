// atlhyper_master/master_model/overview_dto.go
package master_model

import "time"

type OverviewPayload struct {
    GeneratedAt string        `json:"generatedAt"`
    Cards       Cards         `json:"cards"`
    Trends      Trends        `json:"trends"`
    RecentEvents []EventRow   `json:"recentEvents"`
    NodeUsage   NodeUsageBloc `json:"nodeUsage"`
}

type Cards struct {
    ClusterHealth struct {
        Status       string  `json:"status"`
        NodeReadyPct float64 `json:"nodeReadyPct"`
        PodHealthyPct float64 `json:"podHealthyPct"`
    } `json:"clusterHealth"`
    Nodes struct {
        Ready int `json:"ready"`
        Total int `json:"total"`
        ReadyPct float64 `json:"readyPct"`
    } `json:"nodes"`
    CPU struct {
        UsagePct   float64 `json:"usagePct"`
        UsageCores float64 `json:"usageCores"`
        TotalCores float64 `json:"totalCores"`
    } `json:"cpu"`
    Memory struct {
        UsagePct    float64 `json:"usagePct"`
        UsageBytes  uint64  `json:"usageBytes"`
        TotalBytes  uint64  `json:"totalBytes"`
    } `json:"memory"`
    Alerts24h struct {
        Total int `json:"total"`
        Critical int `json:"critical"`
        Warning int `json:"warning"`
        Info int `json:"info"`
    } `json:"alerts24h"`
}

type Trends struct {
    Resource struct {
        WindowMinutes int `json:"windowMinutes"`
        StepSeconds   int `json:"stepSeconds"`
        Series []struct {
            Ts        time.Time `json:"ts"`
            CpuMaxPct float64   `json:"cpuMaxPct"`
            MemMaxPct float64   `json:"memMaxPct"`
            TempMaxC  float64   `json:"tempMaxC"`
        } `json:"series"`
    } `json:"resource"`
    Events struct {
        WindowHours int    `json:"windowHours"`
        Bucket      string `json:"bucket"`
        Series []struct {
            Ts        time.Time `json:"ts"`
            Critical  int       `json:"critical"`
            Warning   int       `json:"warning"`
            Info      int       `json:"info"`
            Total     int       `json:"total"`
        } `json:"series"`
    } `json:"events"`
}

type EventRow struct {
    Time      time.Time `json:"time"`
    Severity  string    `json:"severity"`
    Source    string    `json:"source"`
    Namespace string    `json:"namespace"`
    Message   string    `json:"message"`
    Name      string    `json:"name"`
    Reason    string    `json:"reason"`
}

type NodeUsageBloc struct {
    Metric string `json:"metric"` // "CPU" 或 "Memory"
    Items  []struct {
        Name   string  `json:"name"`
        Role   string  `json:"role,omitempty"`
        Ready  bool    `json:"ready"`
        CpuPct float64 `json:"cpuPct"`
        MemPct float64 `json:"memPct"`
    } `json:"items"`
    Total int `json:"total"`
}
