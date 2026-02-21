package application

import (
	"context"
	"fmt"
	"time"

	"github.com/wyfcoding/financialtrading/internal/marketreplay/domain"
	"github.com/wyfcoding/pkg/messagequeue"
)

// 生成摘要：驱动行情回放引擎。

type ReplayEngine struct {
	mq messagequeue.EventPublisher
}

func NewReplayEngine(mq messagequeue.EventPublisher) *ReplayEngine {
	return &ReplayEngine{mq: mq}
}

// StartReplay 执行回放循环。
func (e *ReplayEngine) StartReplay(ctx context.Context, task *domain.ReplayTask, ticks []domain.MarketTick) {
	if len(ticks) == 0 {
		return
	}

	for i := 0; i < len(ticks); i++ {
		select {
		case <-ctx.Done():
			return
		default:
			// 发布到 Kafka
			key := fmt.Sprintf("%s-%d", ticks[i].Symbol, ticks[i].Timestamp.UnixMilli())
			_ = e.mq.Publish(ctx, task.Topic, key, ticks[i])

			// 计算休眠时间 (受回放倍速影响)
			if i < len(ticks)-1 {
				wait := ticks[i+1].Timestamp.Sub(ticks[i].Timestamp)
				actualWait := time.Duration(float64(wait) / task.SpeedFactor)
				time.Sleep(actualWait)
			}
		}
	}
}
