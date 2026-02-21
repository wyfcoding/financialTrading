// 变更说明：
// Disruptor LMAX 架构引擎封装应用服务。
// 摒弃 Go Channel (因为内部带锁互斥且需要 GC 挂起)。
// 接入 pkg/disruptor/RingBuffer 实现 单生产者-单消费者 (Single Producer - Single Consumer) 零分配架构。
package application

import (
	"context"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	// 使用预写好的 disruptor 无锁高速队列基建
	"github.com/wyfcoding/pkg/disruptor"
)

// RingBuffer 事件槽，对应内存块里的一个坑位
type OrderCommandEvent struct {
	Type      int8 // 1: PlaceOrder, 2: CancelOrder
	OrderID   uint64
	AccountID uint64
	Symbol    string
	IsBuy     bool
	Price     int64
	Qty       int64
	Timestamp int64
}

// 引擎配置常量
// 对齐 2 的 N 次方可以极大优化求余取模(Modulo) 的二进制位运算 & (size-1) 速度
const RingBufferSize int64 = 1024 * 1024 // 104万容量

type LmaxEngine struct {
	ringBuffer     *disruptor.RingBuffer[OrderCommandEvent]
	books          map[string]*domain.FastOrderBook // 交易对 -> 订单薄
	producerSeq    *disruptor.Sequence              // 发布者进度索引
	consumerSeq    *disruptor.Sequence              // 撮合者进度索引
	tradeOutChan   chan []domain.Trade              // 输出结果通道(给后端落地DB落账使用，可以用 MPSC)
	tradePublisher func(ctx context.Context, trades []domain.Trade) error
	logger         *slog.Logger
	isRunning      atomic.Bool
}

func NewLmaxEngine(logger *slog.Logger) *LmaxEngine {
	// 1. 预先在堆内存中申请完全连续的一整块内存槽 (Zero Allocation)
	rb := disruptor.NewRingBuffer[OrderCommandEvent](RingBufferSize, &disruptor.YieldingWaitStrategy{})

	return &LmaxEngine{
		ringBuffer:   rb,
		books:        make(map[string]*domain.FastOrderBook),
		producerSeq:  disruptor.NewSequence(-1),
		consumerSeq:  disruptor.NewSequence(-1),
		tradeOutChan: make(chan []domain.Trade, 65536), // 输出流
		logger:       logger,
	}
}

// SetTradePublisher configures an optional async publisher for matched trades.
func (e *LmaxEngine) SetTradePublisher(publisher func(ctx context.Context, trades []domain.Trade) error) {
	e.tradePublisher = publisher
}

// SubmitOrder (异步高速投递口)  网关只需调用这个，无需阻塞等待！
// 因为是 RingBuffer，不需要做 GC 回收，拿到内存块覆盖旧数据即可。
func (e *LmaxEngine) SubmitOrder(orderID, accID uint64, symbol string, isBuy bool, price, qty int64) {
	// 单生产者路径：顺序推进写入序号，无需额外锁。
	nextSeq := e.producerSeq.Get() + 1
	for nextSeq-e.consumerSeq.Get() >= RingBufferSize {
		runtime.Gosched()
	}
	e.producerSeq.Set(nextSeq)

	// 拿到了自己的私人槽位，直接物理内存覆盖
	event := e.ringBuffer.Get(nextSeq)
	event.Type = 1
	event.OrderID = orderID
	event.AccountID = accID
	event.Symbol = symbol
	event.IsBuy = isBuy
	event.Price = price
	event.Qty = qty
	event.Timestamp = time.Now().UnixNano()
}

// Start 启动撮合宿线 (绑核强力运算线程)
func (e *LmaxEngine) Start(ctx context.Context) {
	if e.isRunning.Swap(true) {
		return
	}
	e.logger.Info("LMAX matching engine core thread started.")

	// 创建后端异步落盘工作者
	go e.journalWorker(ctx)

	// 这是金融系统全域最核心的 Loop
	go func() {
		// runtime.LockOSThread() // 极端场景下可以绑定OS线程禁止切走
		var currentSeq = e.consumerSeq.Get() + 1

		for {
			// 一旦 ctx done 退出
			select {
			case <-ctx.Done():
				e.isRunning.Store(false)
				return
			default:
			}

			// 如果进度还没赶上生产者，证明槽里有数据
			if currentSeq <= e.producerSeq.Get() {
				event := e.ringBuffer.Get(currentSeq)

				// 进入具体撮合逻辑，无锁环境
				e.processEvent(event)

				// 游标往前推一位
				e.consumerSeq.Set(currentSeq)
				currentSeq++
			} else {
				// 没数据，空转。在 HFT 场景往往写为 runtime.Gosched() 而不是 sleep
				runtime.Gosched()
			}
		}
	}()
}

// processEvent 真正的撮合分发（此方法绝对无锁！）
func (e *LmaxEngine) processEvent(event *OrderCommandEvent) {
	book, exists := e.books[event.Symbol]
	if !exists {
		book = domain.NewFastOrderBook(event.Symbol)
		e.books[event.Symbol] = book
	}

	if event.Type == 1 {
		// 将网络事件转换为内存极速订单实体
		fo := domain.FastOrder{
			OrderID:   event.OrderID,
			AccountID: event.AccountID,
			Price:     event.Price,
			Qty:       event.Qty,
			Time:      event.Timestamp,
		}

		side := domain.Sell
		if event.IsBuy {
			side = domain.Buy
		}

		trades := book.Process(fo, side)

		if len(trades) > 0 {
			// 将成交结果推入异步记录通道持久化，不要阻塞撮合进程
			e.tradeOutChan <- trades
		}
	}
	// 取消订单等逻辑...
}

// journalWorker 获取撮合成功的快照进行落盘入库或写入 Kafka
func (e *LmaxEngine) journalWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case trades := <-e.tradeOutChan:
			// 在外层独立协程执行，完全不抢占撮合协程的算力
			if len(trades) > 0 {
				if e.tradePublisher != nil {
					if err := e.tradePublisher(ctx, trades); err != nil {
						e.logger.Error("failed to publish matched trades", "count", len(trades), "error", err)
					}
				} else {
					e.logger.Debug("matched trades ready for downstream publish", "count", len(trades))
				}
			}
		}
	}
}
