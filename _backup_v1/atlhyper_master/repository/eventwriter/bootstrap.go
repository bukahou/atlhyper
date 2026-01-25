// repository/eventwriter/bootstrap.go
// 事件写入调度器
package eventwriter

import (
	"context"
	"log"
	"time"
)

const defaultSyncInterval = 30 * time.Second

// StartLogWriterScheduler 启动事件日志写入调度器
func StartLogWriterScheduler() {
	go func() {
		ticker := time.NewTicker(defaultSyncInterval)
		defer ticker.Stop()

		for range ticker.C {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := SyncEventsToSQL(ctx); err != nil {
				log.Printf("⚠️ 事件同步失败: %v", err)
			}
			cancel()
		}
	}()

	log.Println("✅ 事件日志写入调度器已启动")
}
