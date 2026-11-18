package core

import (
	"testing"
	"time"
)

// 同一事件一分钟内只累计一次；不同事件互不影响；一小时没动过的记录被清掉
func TestSignalThrottle(t *testing.T) {
	var throttle signalThrottle
	t0 := time.Date(2026, 11, 15, 6, 0, 0, 0, time.UTC)

	if !throttle.due("inc-1", t0) {
		t.Fatal("第一次应累计")
	}
	if throttle.due("inc-1", t0.Add(59*time.Second)) {
		t.Error("一分钟内不应再累计")
	}
	if !throttle.due("inc-2", t0.Add(10*time.Second)) {
		t.Error("另一个事件不受影响")
	}
	if !throttle.due("inc-1", t0.Add(time.Minute)) {
		t.Error("满一分钟应再累计")
	}
	throttle.due("inc-3", t0.Add(2*time.Hour))
	if _, kept := throttle.last["inc-2"]; kept {
		t.Error("一小时没动过的记录应被清掉")
	}
}
