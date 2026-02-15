// atlhyper_master_v2/aiops/correlator/serializer.go
// 依赖图 gzip/JSON 序列化/反序列化
package correlator

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Serialize 将图序列化为 gzip(JSON)
func Serialize(graph *aiops.DependencyGraph) ([]byte, error) {
	jsonData, err := json.Marshal(graph)
	if err != nil {
		return nil, fmt.Errorf("marshal graph: %w", err)
	}

	var buf bytes.Buffer
	zw, err := gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	if err != nil {
		return nil, fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err := zw.Write(jsonData); err != nil {
		zw.Close()
		return nil, fmt.Errorf("gzip write: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	return buf.Bytes(), nil
}

// Deserialize 从 gzip(JSON) 恢复图
func Deserialize(data []byte) (*aiops.DependencyGraph, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer zr.Close()

	jsonData, err := io.ReadAll(zr)
	if err != nil {
		return nil, fmt.Errorf("gunzip read: %w", err)
	}

	var graph aiops.DependencyGraph
	if err := json.Unmarshal(jsonData, &graph); err != nil {
		return nil, fmt.Errorf("unmarshal graph: %w", err)
	}
	graph.RebuildIndex()
	return &graph, nil
}
