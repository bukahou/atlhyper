// Package api 内嵌 Master 对 Web 的 OpenAPI 规格（openapi.yaml），供契约测试读取。
package api

import _ "embed"

// OpenAPI 规格原文（YAML）
//
//go:embed openapi.yaml
var OpenAPI []byte
