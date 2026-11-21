package severity

import "testing"

func TestRank(t *testing.T) {
	if !(Rank(Low) < Rank(Medium) && Rank(Medium) < Rank(High) && Rank(High) < Rank(Critical)) {
		t.Fatal("档位应从 low 到 critical 递增")
	}
	if Rank("bogus") != 0 || Rank("") != 0 {
		t.Error("不认识的取值档位应为 0")
	}
	if !AtLeast(High, Medium) || AtLeast(Low, High) || !AtLeast(Low, "") {
		t.Error("AtLeast 结果不对")
	}
}

// 库里的比较与 Go 里的 Rank 是同一张表
func TestRankSQL(t *testing.T) {
	want := "CASE severity WHEN 'critical' THEN 4 WHEN 'high' THEN 3 WHEN 'medium' THEN 2 WHEN 'low' THEN 1 ELSE 0 END"
	if got := RankSQL("severity"); got != want {
		t.Errorf("RankSQL =\n%s\nwant\n%s", got, want)
	}
}
