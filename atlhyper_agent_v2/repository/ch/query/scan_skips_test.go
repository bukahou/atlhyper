package query

import (
	"errors"
	"testing"
)

// 跳过的行要留痕：数量与第一条原因
func TestScanSkips(t *testing.T) {
	s := newScanSkips("probe")
	s.report() // 没有跳过时不报
	s.add(errors.New("column 2: converting Nullable(String) to string"))
	s.add(errors.New("second"))
	if s.count != 2 || s.first == nil || s.first.Error() == "second" {
		t.Errorf("scanSkips = %+v, want 2 行且记下第一条原因", s)
	}
}
