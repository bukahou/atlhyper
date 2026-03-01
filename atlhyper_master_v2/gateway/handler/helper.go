// atlhyper_master_v2/gateway/handler/helper.go
// Handler 公共辅助函数
package handler

import (
	"encoding/json"
	"net/http"
)

// WriteJSON 写入 JSON 响应（导出供子包使用）
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError 写入错误响应（导出供子包使用）
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

// writeJSON 包内便捷别名
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	WriteJSON(w, status, data)
}

// writeError 包内便捷别名
func writeError(w http.ResponseWriter, status int, message string) {
	WriteError(w, status, message)
}
