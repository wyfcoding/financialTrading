// 变更说明：
// 高频撮合极速 gRPC 网关。
// 提供 Zero-Allocation (零分配) 级别的协议转换。
package grpc

import (
	"context"
	"log/slog"

	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/pkg/contextx"
	// pb "github.com/wyfcoding/financialtrading/go-api/matching/v1"
)

// ========= 模拟 protobuf interface ========= //
type FakePlaceOrderRequest struct {
	OrderId   uint64 `protobuf:"varint,1,opt,name=order_id"`
	AccountId uint64 `protobuf:"varint,2,opt,name=account_id"`
	Symbol    string `protobuf:"bytes,3,opt,name=symbol"`
	IsBuy     bool   `protobuf:"varint,4,opt,name=is_buy"`
	Price     int64  `protobuf:"varint,5,opt,name=price"` // 价格已放大10000倍的定点数
	Qty       int64  `protobuf:"varint,6,opt,name=qty"`   // 数量已放大10000倍的定点数
}

type FakePlaceOrderResponse struct {
	Success bool   `protobuf:"varint,1,opt,name=success"`
	TraceId string `protobuf:"bytes,2,opt,name=trace_id"`
}

// ========================================== //

type MatchingGrpcServer struct {
	// pb.UnimplementedMatchingServiceServer
	engine *application.LmaxEngine
	logger *slog.Logger
}

func NewMatchingGrpcServer(engine *application.LmaxEngine, logger *slog.Logger) *MatchingGrpcServer {
	return &MatchingGrpcServer{
		engine: engine,
		logger: logger,
	}
}

// PlaceOrder 极速挂单接口。
// 注意：本接口只有纯粹的参数提取和 RingBuffer 投递，【禁止】任何网络 IO、数据库操作和内存分配！
func (s *MatchingGrpcServer) PlaceOrder(ctx context.Context, req *FakePlaceOrderRequest) (*FakePlaceOrderResponse, error) {
	// 获取链路追踪ID，但不做任何强计算拼接
	traceID := contextx.GetRequestID(ctx)

	// O(1) 乃至纳秒级投递：直接通过 CAS 获取环形队列内存插槽并原地覆盖
	// 没有任何 mutex 锁和 go routine 的挂起！
	s.engine.SubmitOrder(
		req.OrderId,
		req.AccountId,
		req.Symbol,
		req.IsBuy,
		req.Price,
		req.Qty,
	)

	// 直接返回 ACK (Order Accepted)。真正的撮合发生在绑核的另一个死循环单线程中。
	return &FakePlaceOrderResponse{
		Success: true,
		TraceId: traceID,
	}, nil
}
