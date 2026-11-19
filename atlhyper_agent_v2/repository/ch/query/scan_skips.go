package query

import "AtlHyper/common/logger"

// scanSkips 一次查询里解析失败、被跳过的结果行。一行坏数据不让整次查询失败，
// 但必须留痕（数量 + 第一条原因）：否则列类型一变，数据就悄悄少了一截而没人知道
type scanSkips struct {
	query string
	count int
	first error
}

func newScanSkips(query string) *scanSkips { return &scanSkips{query: query} }

func (s *scanSkips) add(err error) {
	if s.first == nil {
		s.first = err
	}
	s.count++
}

// report 有跳过的行时记一条告警；在查询函数里 defer 调用
func (s *scanSkips) report() {
	if s.count > 0 {
		logger.Warn("ClickHouse 结果行解析失败，已跳过", "query", s.query, "rows", s.count, "err", s.first)
	}
}
