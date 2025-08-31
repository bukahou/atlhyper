package overview

import (
	"strconv"
	"strings"
	"time"
)

func floorToBucket(t time.Time, bucket time.Duration) time.Time {
	sec := int64(bucket.Seconds())
	return time.Unix((t.Unix()/sec)*sec, 0).UTC()
}


// parseMilliCPU: "216m" → 0.216, "8" → 8
func parseMilliCPU(v string) float64 {
	if strings.HasSuffix(v, "m") {
		num, _ := strconv.ParseFloat(strings.TrimSuffix(v, "m"), 64)
		return num / 1000
	}
	num, _ := strconv.ParseFloat(v, 64)
	return num
}

// parseCPU: "8" → 8
func parseCPU(v string) float64 {
	num, _ := strconv.ParseFloat(v, 64)
	return num
}

// parseKiToBytes: "4844752Ki" → bytes
func parseKiToBytes(v string) float64 {
	if strings.HasSuffix(v, "Ki") {
		num, _ := strconv.ParseFloat(strings.TrimSuffix(v, "Ki"), 64)
		return num * 1024
	}
	num, _ := strconv.ParseFloat(v, 64)
	return num
}