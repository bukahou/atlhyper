package llmtest

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// packageOf 客户端类型所在的包：适配层自己起的 goroutine，栈里有这个包的函数
func packageOf(client llm.LLMClient) string {
	typ := reflect.TypeOf(client)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	return typ.PkgPath()
}

// goroutinesIn 栈里有 pkg 包函数的 goroutine，返回各自的栈（只用于失败时给人看）。
// 用运行时的结构化栈记录、按函数名前缀判断，不解析栈文本；记录只保留栈顶 32 帧，
// 卡在适配层自己往通道发送上的 goroutine 栈很浅，认得出。
func goroutinesIn(pkg string) []string {
	records := make([]runtime.StackRecord, runtime.NumGoroutine()+16)
	n, ok := runtime.GoroutineProfile(records)
	for !ok {
		records = make([]runtime.StackRecord, n+16)
		n, ok = runtime.GoroutineProfile(records)
	}
	prefix := pkg + "."
	var stacks []string
	for _, record := range records[:n] {
		var stack strings.Builder
		matched := false
		frames := runtime.CallersFrames(record.Stack())
		for {
			frame, more := frames.Next()
			matched = matched || strings.HasPrefix(frame.Function, prefix)
			fmt.Fprintf(&stack, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
			if !more {
				break
			}
		}
		if matched {
			stacks = append(stacks, stack.String())
		}
	}
	return stacks
}

// settledGoroutinesIn 等上一个用例的 goroutine 退干净再数（最多等 timeout），作为本用例的基线
func settledGoroutinesIn(pkg string, timeout time.Duration) int {
	deadline := time.Now().Add(timeout)
	for {
		count := len(goroutinesIn(pkg))
		if count == 0 || time.Now().After(deadline) {
			return count
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// waitGoroutinesIn 等 pkg 包里的 goroutine 降回 baseline 以内；超时返回还在的那些的栈，按时退出返回 nil
func waitGoroutinesIn(pkg string, baseline int, timeout time.Duration) []string {
	deadline := time.Now().Add(timeout)
	for {
		stacks := goroutinesIn(pkg)
		if len(stacks) <= baseline {
			return nil
		}
		if time.Now().After(deadline) {
			return stacks
		}
		time.Sleep(10 * time.Millisecond)
	}
}
