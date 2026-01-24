// atlhyper_master_v2/mq/redis/bus.go
// RedisBus Redis 消息队列实现
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"

	"AtlHyper/atlhyper_master_v2/model"
)

// Key 前缀
const (
	keyQueue  = "mq:queue:"  // + clusterID:topic -> LIST
	keyCmd    = "mq:cmd:"    // + cmdID -> JSON (CommandStatus)
	keyResult = "mq:result:" // + cmdID -> LIST (blocking wait signal)
)

// Config RedisBus 配置
type Config struct {
	Addr     string
	Password string
	DB       int
}

// RedisBus Redis 消息队列
type RedisBus struct {
	client *redis.Client
}

// New 创建 RedisBus
func New(cfg Config) *RedisBus {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &RedisBus{client: client}
}

// Start 启动 RedisBus
func (b *RedisBus) Start() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := b.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	log.Println("[RedisBus] 已启动")
	return nil
}

// Stop 停止 RedisBus
func (b *RedisBus) Stop() error {
	if err := b.client.Close(); err != nil {
		return fmt.Errorf("redis close failed: %w", err)
	}
	log.Println("[RedisBus] 已停止")
	return nil
}

// queueKey 生成队列 key
func queueKey(clusterID, topic string) string {
	return keyQueue + clusterID + ":" + topic
}

// EnqueueCommand 入队指令到指定 topic
func (b *RedisBus) EnqueueCommand(clusterID, topic string, cmd *model.Command) error {
	ctx := context.Background()

	// 序列化指令
	cmdData, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	// 入队（LPUSH，BRPOP 从右取）
	if err := b.client.LPush(ctx, queueKey(clusterID, topic), cmdData).Err(); err != nil {
		return fmt.Errorf("lpush command: %w", err)
	}

	// 记录指令状态
	status := &model.CommandStatus{
		CommandID: cmd.ID,
		Status:    model.CommandStatusPending,
		CreatedAt: cmd.CreatedAt,
	}
	statusData, _ := json.Marshal(status)
	// 状态保留 24h
	b.client.Set(ctx, keyCmd+cmd.ID, statusData, 24*time.Hour)

	log.Printf("[RedisBus] 指令已入队: %s -> %s [%s]", cmd.ID, clusterID, topic)
	return nil
}

// WaitCommand 等待指定 topic 的指令（阻塞等待）
func (b *RedisBus) WaitCommand(ctx context.Context, clusterID, topic string, timeout time.Duration) (*model.Command, error) {
	// BRPOP 阻塞等待
	result, err := b.client.BRPop(ctx, timeout, queueKey(clusterID, topic)).Result()
	if err == redis.Nil {
		return nil, nil // 超时
	}
	if err != nil {
		// context cancelled
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("brpop: %w", err)
	}

	// result[0] = key, result[1] = value
	var cmd model.Command
	if err := json.Unmarshal([]byte(result[1]), &cmd); err != nil {
		return nil, fmt.Errorf("unmarshal command: %w", err)
	}

	// 更新状态为 running
	b.updateCommandStatus(cmd.ID, model.CommandStatusRunning)

	return &cmd, nil
}

// updateCommandStatus 更新指令状态
func (b *RedisBus) updateCommandStatus(cmdID, status string) {
	ctx := context.Background()

	data, err := b.client.Get(ctx, keyCmd+cmdID).Bytes()
	if err != nil {
		return
	}

	var cs model.CommandStatus
	if json.Unmarshal(data, &cs) != nil {
		return
	}

	cs.Status = status
	if status == model.CommandStatusRunning {
		now := time.Now()
		cs.StartedAt = &now
	}

	newData, _ := json.Marshal(&cs)
	b.client.Set(ctx, keyCmd+cmdID, newData, 24*time.Hour)
}

// AckCommand 确认指令完成
func (b *RedisBus) AckCommand(cmdID string, result *model.CommandResult) error {
	ctx := context.Background()

	data, err := b.client.Get(ctx, keyCmd+cmdID).Bytes()
	if err != nil {
		return nil // 指令不存在，忽略
	}

	var cs model.CommandStatus
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil
	}

	now := time.Now()
	cs.FinishedAt = &now
	cs.Result = result
	if result.Success {
		cs.Status = model.CommandStatusSuccess
	} else {
		cs.Status = model.CommandStatusFailed
	}

	newData, _ := json.Marshal(&cs)
	b.client.Set(ctx, keyCmd+cmdID, newData, 24*time.Hour)

	// 通知等待者（LPUSH 到 result list）
	resultData, _ := json.Marshal(result)
	b.client.LPush(ctx, keyResult+cmdID, resultData)
	// result key 保留较短时间
	b.client.Expire(ctx, keyResult+cmdID, 10*time.Minute)

	log.Printf("[RedisBus] 指令已完成: %s -> %s", cmdID, cs.Status)
	return nil
}

// GetCommandStatus 获取指令状态
func (b *RedisBus) GetCommandStatus(cmdID string) (*model.CommandStatus, error) {
	ctx := context.Background()

	data, err := b.client.Get(ctx, keyCmd+cmdID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get command status: %w", err)
	}

	var cs model.CommandStatus
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil, fmt.Errorf("unmarshal status: %w", err)
	}
	return &cs, nil
}

// WaitCommandResult 等待指令执行完成（阻塞等待）
func (b *RedisBus) WaitCommandResult(cmdID string, timeout time.Duration) (*model.CommandResult, error) {
	ctx := context.Background()

	// 先检查是否已完成
	data, err := b.client.Get(ctx, keyCmd+cmdID).Bytes()
	if err == nil {
		var cs model.CommandStatus
		if json.Unmarshal(data, &cs) == nil && cs.Result != nil {
			return cs.Result, nil
		}
	}

	// BRPOP 等待结果
	result, err := b.client.BRPop(ctx, timeout, keyResult+cmdID).Result()
	if err == redis.Nil {
		return nil, nil // 超时
	}
	if err != nil {
		return nil, fmt.Errorf("brpop result: %w", err)
	}

	var cmdResult model.CommandResult
	if err := json.Unmarshal([]byte(result[1]), &cmdResult); err != nil {
		return nil, fmt.Errorf("unmarshal result: %w", err)
	}
	return &cmdResult, nil
}
