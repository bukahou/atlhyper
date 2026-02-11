package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CounterRate_Normal(t *testing.T) {
	assert.InDelta(t, 10.0, counterRate(200, 50, 15), 0.01) // (200-50)/15
}

func Test_CounterRate_Reset(t *testing.T) {
	assert.InDelta(t, 0.0, counterRate(10, 100, 15), 0.01) // cur < prev → 0
}

func Test_CounterRate_ZeroElapsed(t *testing.T) {
	assert.InDelta(t, 0.0, counterRate(200, 50, 0), 0.01) // elapsed=0 → 0
}

func Test_CounterRate_NegativeElapsed(t *testing.T) {
	assert.InDelta(t, 0.0, counterRate(200, 50, -1), 0.01)
}

func Test_CounterDelta_Normal(t *testing.T) {
	assert.InDelta(t, 150.0, counterDelta(200, 50), 0.01)
}

func Test_CounterDelta_Reset(t *testing.T) {
	assert.InDelta(t, 0.0, counterDelta(10, 100), 0.01) // cur < prev → 0
}

func Test_CounterDelta_Same(t *testing.T) {
	assert.InDelta(t, 0.0, counterDelta(100, 100), 0.01)
}
