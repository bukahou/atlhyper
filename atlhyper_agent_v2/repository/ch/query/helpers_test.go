package query

import (
	"math"
	"testing"
	"time"
)

// =============================================================================
// TestComputeRateSeries
// =============================================================================

func TestComputeRateSeries_Normal(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []rawPoint{
		{Time: base, Value: 100},
		{Time: base.Add(10 * time.Second), Value: 200},
		{Time: base.Add(20 * time.Second), Value: 400},
	}

	result := computeRateSeries(points)

	if len(result) != 2 {
		t.Fatalf("expected 2 result points, got %d", len(result))
	}
	// (200-100)/10 = 10
	if result[0].Value != 10 {
		t.Errorf("expected rate 10, got %f", result[0].Value)
	}
	// (400-200)/10 = 20
	if result[1].Value != 20 {
		t.Errorf("expected rate 20, got %f", result[1].Value)
	}
}

func TestComputeRateSeries_CounterReset(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	points := []rawPoint{
		{Time: base, Value: 500},
		{Time: base.Add(10 * time.Second), Value: 50}, // counter reset
	}

	result := computeRateSeries(points)

	if len(result) != 1 {
		t.Fatalf("expected 1 result point, got %d", len(result))
	}
	// delta < 0, use 50 as delta -> 50/10 = 5
	if result[0].Value != 5 {
		t.Errorf("expected rate 5, got %f", result[0].Value)
	}
}

func TestComputeRateSeries_TooFewPoints(t *testing.T) {
	result := computeRateSeries(nil)
	if result != nil {
		t.Error("expected nil for nil input")
	}

	result = computeRateSeries([]rawPoint{{Time: time.Now(), Value: 1}})
	if result != nil {
		t.Error("expected nil for single point")
	}
}

func TestComputeRateSeries_SameTimestamp(t *testing.T) {
	now := time.Now()
	points := []rawPoint{
		{Time: now, Value: 100},
		{Time: now, Value: 200}, // dt=0, should skip
	}

	result := computeRateSeries(points)

	if len(result) != 0 {
		t.Errorf("expected 0 points for same timestamp, got %d", len(result))
	}
}

// =============================================================================
// TestComputeRate
// =============================================================================

func TestComputeRate_Normal(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(60 * time.Second)

	rate := computeRate(100, 160, t1, t2)
	if rate != 1 {
		t.Errorf("expected rate 1, got %f", rate)
	}
}

func TestComputeRate_ZeroDuration(t *testing.T) {
	now := time.Now()
	rate := computeRate(0, 100, now, now)
	if rate != 0 {
		t.Errorf("expected rate 0 for zero duration, got %f", rate)
	}
}

func TestComputeRate_CounterReset(t *testing.T) {
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(10 * time.Second)

	rate := computeRate(500, 30, t1, t2)
	// delta < 0 => use 30 as delta => 30/10 = 3
	if rate != 3 {
		t.Errorf("expected rate 3, got %f", rate)
	}
}

// =============================================================================
// TestHistogramPercentile
// =============================================================================

func TestHistogramPercentile_P50(t *testing.T) {
	// Buckets: [0, 10), [10, 50), [50, 100), [100, +Inf)
	bounds := []float64{10, 50, 100}
	counts := []uint64{20, 30, 40, 10} // total=100

	p50 := histogramPercentile(bounds, counts, 0.50)

	// P50: target = 50
	// bucket 0: cumulative=20 < 50
	// bucket 1: cumulative=50 >= 50
	// lower=10, upper=50, prevCumulative=20, c=30
	// fraction = (50-20)/30 = 1.0
	// result = 10 + 1.0 * (50-10) = 50
	if p50 != 50 {
		t.Errorf("expected P50=50, got %f", p50)
	}
}

func TestHistogramPercentile_P99(t *testing.T) {
	bounds := []float64{10, 50, 100}
	counts := []uint64{20, 30, 40, 10} // total=100

	p99 := histogramPercentile(bounds, counts, 0.99)

	// target = 99
	// bucket 0: 20 < 99
	// bucket 1: 50 < 99
	// bucket 2: 90 < 99
	// bucket 3 (Inf): 100 >= 99 -> return bounds[last]=100
	if p99 != 100 {
		t.Errorf("expected P99=100, got %f", p99)
	}
}

func TestHistogramPercentile_Empty(t *testing.T) {
	v := histogramPercentile(nil, nil, 0.5)
	if v != 0 {
		t.Errorf("expected 0 for empty, got %f", v)
	}

	v = histogramPercentile([]float64{10}, []uint64{0, 0}, 0.5)
	if v != 0 {
		t.Errorf("expected 0 for zero total, got %f", v)
	}
}

func TestHistogramPercentile_Boundary(t *testing.T) {
	bounds := []float64{10, 50}
	counts := []uint64{10, 0, 0}

	// All in first bucket, p=0
	v := histogramPercentile(bounds, counts, 0)
	if v != 0 {
		t.Errorf("expected 0 for p=0, got %f", v)
	}

	// p=1
	v = histogramPercentile(bounds, counts, 1)
	if v != 50 {
		t.Errorf("expected 50 for p=1, got %f", v)
	}
}

// =============================================================================
// TestSafeDiv / SafeDivPct
// =============================================================================

func TestSafeDiv(t *testing.T) {
	tests := []struct {
		name     string
		a, b     float64
		expected float64
	}{
		{"Normal", 10, 5, 2},
		{"ZeroDenominator", 10, 0, 0},
		{"ZeroBoth", 0, 0, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := safeDiv(tc.a, tc.b)
			if got != tc.expected {
				t.Errorf("safeDiv(%f, %f) = %f, want %f", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}

func TestSafeDivPct(t *testing.T) {
	got := safeDivPct(3, 4)
	if got != 75 {
		t.Errorf("expected 75, got %f", got)
	}
	got = safeDivPct(1, 0)
	if got != 0 {
		t.Errorf("expected 0, got %f", got)
	}
}

// =============================================================================
// TestClamp / RoundTo / SinceSeconds
// =============================================================================

func TestClamp(t *testing.T) {
	if clamp(5, 0, 10) != 5 {
		t.Error("in range value should be unchanged")
	}
	if clamp(-1, 0, 10) != 0 {
		t.Error("below min should return min")
	}
	if clamp(15, 0, 10) != 10 {
		t.Error("above max should return max")
	}
}

func TestRoundTo(t *testing.T) {
	v := roundTo(3.14159, 2)
	if math.Abs(v-3.14) > 0.001 {
		t.Errorf("expected 3.14, got %f", v)
	}
}

func TestSinceSeconds(t *testing.T) {
	if sinceSeconds(5*time.Minute) != 300 {
		t.Error("5m should be 300s")
	}
	if sinceSeconds(0) != 300 {
		t.Error("0 should default to 300s")
	}
	if sinceSeconds(-1*time.Minute) != 300 {
		t.Error("negative should default to 300s")
	}
}

// =============================================================================
// TestParseDurationNanos
// =============================================================================

func TestParseDurationNanos(t *testing.T) {
	// 1 second = 1e9 nanos = 1000 ms
	v := parseDurationNanos(1_000_000_000)
	if v != 1000 {
		t.Errorf("expected 1000ms, got %f", v)
	}

	// 500 microseconds = 500000 nanos = 0.5 ms
	v = parseDurationNanos(500_000)
	if v != 0.5 {
		t.Errorf("expected 0.5ms, got %f", v)
	}
}
