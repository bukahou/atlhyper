package utils

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ReadFileString 读取文件内容为字符串
func ReadFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// ReadFileUint64 读取文件内容为 uint64
func ReadFileUint64(path string) (uint64, error) {
	s, err := ReadFileString(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(s, 10, 64)
}

// ReadFileFloat64 读取文件内容为 float64
func ReadFileFloat64(path string) (float64, error) {
	s, err := ReadFileString(path)
	if err != nil {
		return 0, err
	}
	return strconv.ParseFloat(s, 64)
}

// ReadFileLines 读取文件内容为行数组
func ReadFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// ParseKeyValue 解析 key:value 格式
// 示例: "MemTotal:       16384000 kB"
func ParseKeyValue(line string) (key string, value string) {
	parts := strings.SplitN(line, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

// ParseMemValue 解析内存值（支持 kB 后缀）
// 示例: "16384000 kB" -> 16777216000 (bytes)
func ParseMemValue(s string) int64 {
	s = strings.TrimSpace(s)
	// 移除 kB 后缀
	s = strings.TrimSuffix(s, " kB")
	s = strings.TrimSuffix(s, "kB")

	val, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return 0
	}
	// kB -> bytes
	return val * 1024
}

// ListDirs 列出目录下的所有子目录（包括指向目录的符号链接）
func ListDirs(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var dirs []string
	for _, entry := range entries {
		// 直接是目录
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
			continue
		}
		// 符号链接：检查目标是否是目录
		if entry.Type()&os.ModeSymlink != 0 {
			fullPath := filepath.Join(path, entry.Name())
			info, err := os.Stat(fullPath) // Stat 跟随符号链接
			if err == nil && info.IsDir() {
				dirs = append(dirs, entry.Name())
			}
		}
	}
	return dirs, nil
}

// ListNumericDirs 列出目录下的所有数字命名的子目录（如 /proc 下的 PID）
func ListNumericDirs(path string) ([]int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var pids []int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

// GlobFiles 使用通配符匹配文件
func GlobFiles(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

// FileExists 检查文件是否存在
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// ParseStatFields 解析空格分隔的字段
func ParseStatFields(line string) []string {
	return strings.Fields(line)
}

// Clamp 限制值在指定范围内
func Clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
