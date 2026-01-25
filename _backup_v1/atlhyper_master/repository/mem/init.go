// repository/mem/init.go
// 内存仓库初始化
package mem

import (
	"log"

	"AtlHyper/atlhyper_master/repository"
)

// Init 初始化内存仓库并注册到全局
func Init() {
	repository.InitMem(&HubReader{}, &HubWriter{})
	log.Println("✅ 内存仓库初始化完成")
}
