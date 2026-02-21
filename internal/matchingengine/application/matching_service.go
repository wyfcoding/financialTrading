package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/infrastructure/memory"
)

// MatchingEngine 核心撮合引擎
// 采用 LMAX 架构思想：单线程处理核心逻辑，通过 Channel (模拟 RingBuffer) 输入命令
type MatchingEngine struct {
	orderBooks map[string]*memory.InMemoryOrderBook
	inputChan  chan *inputCommand // 输入队列
	outputChan chan *domain.Trade // 输出队列(成交回报)
}

type commandType int

const (
	cmdPlaceOrder  commandType = 1
	cmdCancelOrder commandType = 2
)

type inputCommand struct {
	Type       commandType
	Order      *domain.Order
	ResultChan chan error // 同步等待结果 (高性能场景可改为异步回调)
}

func NewMatchingEngine() *MatchingEngine {
	return &MatchingEngine{
		orderBooks: make(map[string]*memory.InMemoryOrderBook),
		inputChan:  make(chan *inputCommand, 1024*1024), // 1M buffer for high throughput
		outputChan: make(chan *domain.Trade, 1024*1024),
	}
}

// Start 启动引擎消费线程
func (e *MatchingEngine) Start(ctx context.Context) {
	fmt.Println("🚀 Matching Engine Started...")
	go e.runLoop(ctx)
	go e.consumeTrades(ctx) // 启动一个协程处理成交结果日志
}

// runLoop 是唯一的 Write Thread，不需要加锁
func (e *MatchingEngine) runLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case cmd := <-e.inputChan:
			e.processCommand(cmd)
		}
	}
}

func (e *MatchingEngine) processCommand(cmd *inputCommand) {
	// 懒加载 OrderBook
	symbol := cmd.Order.Symbol
	ob, ok := e.orderBooks[symbol]
	if !ok {
		ob = memory.NewOrderBook(symbol)
		e.orderBooks[symbol] = ob
		fmt.Printf("Created new OrderBook for %s\n", symbol)
	}

	var err error
	var trades []*domain.Trade

	switch cmd.Type {
	case cmdPlaceOrder:
		trades, err = ob.AddOrder(cmd.Order)
	case cmdCancelOrder:
		// Cancel 需要 ID，这里复用 Order 结构传递 ID
		_, err = ob.CancelOrder(cmd.Order.ID)
	}

	// 1. 返回同步结果 (下单是否成功进入撮合)
	if cmd.ResultChan != nil {
		cmd.ResultChan <- err
		close(cmd.ResultChan)
	}

	// 2. 广播成交事件 (异步)
	if len(trades) > 0 {
		for _, t := range trades {
			select {
			case e.outputChan <- t:
			default:
				// 生产环境绝对不能丢弃 Trade，这里应写入 WAL 或 阻塞
				fmt.Printf("CRITICAL: Output channel full! Trade dropped: %v\n", t)
			}
		}
	}
}

// PlaceOrder API: 提交订单
func (e *MatchingEngine) PlaceOrder(order *domain.Order) error {
	resChan := make(chan error, 1)
	cmd := &inputCommand{
		Type:       cmdPlaceOrder,
		Order:      order,
		ResultChan: resChan,
	}

	// 写入队列 (非阻塞写入最好，但这里为了保证处理，可能会阻塞)
	select {
	case e.inputChan <- cmd:
	case <-time.After(1 * time.Second):
		return errors.New("engine overloaded")
	}

	// 等待处理已接受
	return <-resChan
}

// CancelOrder API: 取消订单
func (e *MatchingEngine) CancelOrder(symbol string, orderID uint64) error {
	resChan := make(chan error, 1)
	cmd := &inputCommand{
		Type: cmdCancelOrder,
		Order: &domain.Order{
			ID:     orderID,
			Symbol: symbol,
		},
		ResultChan: resChan,
	}
	e.inputChan <- cmd
	return <-resChan
}

// consumeTrades 简单打印成交记录，实际应写入 DB 或 Kafka
func (e *MatchingEngine) consumeTrades(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-e.outputChan:
			fmt.Printf("✅ TRADE EXECUTED: [%s] %f @ %f (Maker: %d, Taker: %d)\n",
				t.Symbol, t.Quantity, t.Price, t.MakerOrders[0], t.TakerOrder)
		}
	}
}
