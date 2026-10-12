package command

import "encoding/json"

// Metadata 返回去掉 Output 的副本。
// Output 是读类指令的原始内容（Secret / ConfigMap 数据、日志正文），
// 只交给同步等待方，不得落库或经指令查询接口返回。值接收者：不改动原值。
func (r Result) Metadata() Result {
	r.Output = ""
	return r
}

// RedactStoredResult 处理已落库的结果 JSON（早期版本写入的行可能含 output）。
// 解析失败返回空串：宁可丢掉执行结论，也不原样吐出无法确认内容的字段。
func RedactStoredResult(stored string) string {
	if stored == "" {
		return ""
	}
	var r Result
	if err := json.Unmarshal([]byte(stored), &r); err != nil {
		return ""
	}
	b, err := json.Marshal(r.Metadata())
	if err != nil {
		return ""
	}
	return string(b)
}
