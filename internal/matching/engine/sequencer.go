package engine

import (
	"context"

	"github.com/wyfcoding/pkg/algorithm"
)

// OrderEvent 包装订单及其元数据
type OrderEvent struct {
	OrderID string
	Symbol  string
	Price   float64
	Qty     float64
	Side    string // BUY/SELL
}

// OrderSequencer 使用 MpscRingBuffer 对订单进行定序
// 多个生产者（接收 gRPC 请求的 handler）写入，单个消费者（撮合引擎）读取
type OrderSequencer struct {
	buffer *algorithm.MpscRingBuffer[OrderEvent]
	stopCh chan struct{}
}

// NewOrderSequencer 创建定序器
func NewOrderSequencer(capacity uint64) (*OrderSequencer, error) {
	rb, err := algorithm.NewMpscRingBuffer[OrderEvent](capacity)
	if err != nil {
		return nil, err
	}
	return &OrderSequencer{
		buffer: rb,
		stopCh: make(chan struct{}),
	}, nil
}

// Enqueue 生产者：多个 gRPC handler 调用
func (s *OrderSequencer) Enqueue(event *OrderEvent) bool {
	return s.buffer.Offer(event)
}

// Start 消费者：启动单线程定序处理逻辑
func (s *OrderSequencer) Start(ctx context.Context, handler func(*OrderEvent)) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-s.stopCh:
				return
			default:
				// 尝试从 RingBuffer 获取
				event := s.buffer.Poll()
				if event != nil {
					handler(event)
				} else {
					// 如果没有数据，稍微让出 CPU，避免空转消耗过高
					// 在极致高性能场景下会使用 Busy-Wait
					// runtime.Gosched()
				}
			}
		}
	}()
}

func (s *OrderSequencer) Stop() {
	close(s.stopCh)
}
