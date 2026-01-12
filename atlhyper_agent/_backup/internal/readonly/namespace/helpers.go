package namespace

import "math"

type agg struct {
    cpuUsageCores float64
    memUsageBytes float64

    cpuReqCores float64
    cpuLimCores float64
    memReqBytes float64
    memLimBytes float64
}

func ensureAgg(m map[string]*agg, ns string) *agg {
    if a := m[ns]; a != nil {
        return a
    }
    a := &agg{}
    m[ns] = a
    return a
}

func calcUtilPct(usage, limit, request float64) (pct float64, basis string) {
    den := 0.0
    switch {
    case limit > 0:
        den = limit
        basis = "limit"
    case request > 0:
        den = request
        basis = "request"
    default:
        return 0, ""
    }
    p := usage / den * 100
    if p < 0 { p = 0 } else if p > 100 { p = 100 }
    return math.Round(p*10) / 10, basis
}
