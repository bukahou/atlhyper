package slo

import (
	"errors"
	"testing"
)

func TestCheckWindow(t *testing.T) {
	for _, w := range Windows {
		if err := CheckWindow(w); err != nil {
			t.Errorf("%q 应是合法窗口: %v", w, err)
		}
	}
	for _, bad := range []string{"1d", "30d", "15min", "", "7D"} {
		if err := CheckWindow(bad); !errors.Is(err, ErrUnknownWindow) {
			t.Errorf("%q 应返回 ErrUnknownWindow，实际 %v", bad, err)
		}
	}
	if CheckWindow(DefaultWindow) != nil {
		t.Error("默认窗口必须是合法窗口")
	}
}
