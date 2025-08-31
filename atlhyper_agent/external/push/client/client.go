package client

import (
	"context"

	push "AtlHyper/atlhyper_agent/external/push/config"
)

// Sender 统一发送接口：推送器只需传 payload。
type Sender interface {
	Post(ctx context.Context, payload any) (status int, resp []byte, err error)
}

// NewSender 目前固定返回 REST 实现；未来可切换 gRPC。
func NewSender(cfg push.RestClientConfig) Sender {
	return NewRestfulClient(cfg)
}
