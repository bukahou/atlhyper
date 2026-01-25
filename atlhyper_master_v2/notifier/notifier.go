// atlhyper_master_v2/notifier/notifier.go
// 通知模块入口
//
// 包结构:
//
//	notifier/
//	├── notifier.go    -- 包文档
//	├── errors.go      -- 错误定义
//	├── types.go       -- 数据模型 (Alert, Message, AlertSummary)
//	├── channel/       -- 通知渠道适配器
//	│   ├── channel.go -- Channel 接口
//	│   ├── slack.go   -- Slack 实现
//	│   └── email.go   -- Email 实现
//	└── manager/       -- 核心管理逻辑
//	    ├── manager.go -- AlertManager 核心
//	    ├── dedup.go   -- 去重缓存
//	    ├── buffer.go  -- 聚合缓冲
//	    └── limiter.go -- 速率限制
package notifier
