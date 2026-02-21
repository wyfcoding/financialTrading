package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shopspring/decimal"
	pb "github.com/wyfcoding/financialtrading/go-api/matchingengine/v1"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/domain"
	"github.com/wyfcoding/pkg/algos/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type matchingServer struct {
	pb.UnimplementedMatchingEngineServiceServer
	engine *domain.MatchingEngine

	mu     sync.RWMutex
	trades []*types.Trade
}

func newMatchingServer(engine *domain.MatchingEngine) *matchingServer {
	return &matchingServer{
		engine: engine,
		trades: make([]*types.Trade, 0, 1024),
	}
}

func (s *matchingServer) SubmitOrder(ctx context.Context, req *pb.SubmitOrderRequest) (*pb.SubmitOrderResponse, error) {
	_ = ctx
	order, err := toDomainOrder(req)
	if err != nil {
		return nil, err
	}

	result, err := s.engine.SubmitOrder(order)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "submit order failed: %v", err)
	}

	s.appendTrades(result.Trades)
	return &pb.SubmitOrderResponse{
		OrderId:           result.OrderID,
		MatchedTrades:     toProtoTrades(result.Trades),
		RemainingQuantity: result.RemainingQuantity.String(),
		Status:            result.Status,
	}, nil
}

func (s *matchingServer) BatchSubmitOrder(ctx context.Context, req *pb.BatchSubmitOrderRequest) (*pb.BatchSubmitOrderResponse, error) {
	results := make([]*pb.SubmitOrderResponse, 0, len(req.GetOrders()))
	for _, orderReq := range req.GetOrders() {
		res, err := s.SubmitOrder(ctx, orderReq)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	return &pb.BatchSubmitOrderResponse{Results: results}, nil
}

func (s *matchingServer) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.CancelOrderResponse, error) {
	_ = ctx
	side, err := parseSide(req.GetSide())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid side: %v", err)
	}

	result, err := s.engine.CancelOrder(&domain.CancelRequest{
		OrderID:   req.GetOrderId(),
		Symbol:    req.GetSymbol(),
		Side:      side,
		Timestamp: time.Now().UnixNano(),
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cancel order failed: %v", err)
	}

	return &pb.CancelOrderResponse{
		OrderId: result.OrderID,
		Success: result.Success,
		Status:  result.Status,
	}, nil
}

func (s *matchingServer) ExecuteAuction(ctx context.Context, req *pb.ExecuteAuctionRequest) (*pb.ExecuteAuctionResponse, error) {
	_ = ctx
	result, err := s.engine.ExecuteAuction()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "execute auction failed: %v", err)
	}

	s.appendTrades(result.Trades)
	return &pb.ExecuteAuctionResponse{
		Symbol:           req.GetSymbol(),
		EquilibriumPrice: result.EquilibriumPrice.String(),
		MatchedQuantity:  result.MatchedQuantity.String(),
		Trades:           toProtoTrades(result.Trades),
	}, nil
}

func (s *matchingServer) GetOrderBook(ctx context.Context, req *pb.GetOrderBookRequest) (*pb.GetOrderBookResponse, error) {
	_ = ctx
	depth := int(req.GetDepth())
	if depth <= 0 {
		depth = 20
	}

	snapshot := s.engine.GetOrderBookSnapshot(depth)
	return &pb.GetOrderBookResponse{
		Symbol:    snapshot.Symbol,
		Bids:      toProtoLevels(snapshot.Bids),
		Asks:      toProtoLevels(snapshot.Asks),
		Timestamp: snapshot.Timestamp,
	}, nil
}

func (s *matchingServer) GetTrades(ctx context.Context, req *pb.GetTradesRequest) (*pb.GetTradesResponse, error) {
	limit := int(req.GetLimit())
	if limit <= 0 {
		limit = 100
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := make([]*types.Trade, 0, limit)
	for i := len(s.trades) - 1; i >= 0 && len(filtered) < limit; i-- {
		t := s.trades[i]
		if req.GetSymbol() == "" || strings.EqualFold(t.Symbol, req.GetSymbol()) {
			filtered = append(filtered, t)
		}
	}

	return &pb.GetTradesResponse{
		Symbol: req.GetSymbol(),
		Trades: toProtoTrades(filtered),
	}, nil
}

func (s *matchingServer) appendTrades(trades []*types.Trade) {
	if len(trades) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.trades = append(s.trades, trades...)
	const maxTrades = 10000
	if len(s.trades) > maxTrades {
		s.trades = s.trades[len(s.trades)-maxTrades:]
	}
}

func toDomainOrder(req *pb.SubmitOrderRequest) (*types.Order, error) {
	side, err := parseSide(req.GetSide())
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid side: %v", err)
	}

	price, err := decimalFromString("price", req.GetPrice())
	if err != nil {
		return nil, err
	}
	quantity, err := decimalFromString("quantity", req.GetQuantity())
	if err != nil {
		return nil, err
	}

	orderID := req.GetOrderId()
	if orderID == "" {
		orderID = fmt.Sprintf("ME-%d", time.Now().UnixNano())
	}

	order := &types.Order{
		OrderID:     orderID,
		UserID:      req.GetUserId(),
		Symbol:      req.GetSymbol(),
		Side:        side,
		Price:       price,
		Quantity:    quantity,
		Timestamp:   time.Now().UnixNano(),
		IsIceberg:   req.GetIsIceberg(),
		PostOnly:    req.GetPostOnly(),
		OrderType:   types.OrderTypeLimit,
		TimeInForce: types.TIFGTC,
	}

	if req.GetIsIceberg() && req.GetIcebergDisplayQuantity() != "" {
		displayQty, err := decimalFromString("iceberg_display_quantity", req.GetIcebergDisplayQuantity())
		if err != nil {
			return nil, err
		}
		order.DisplayQty = displayQty
	}

	return order, nil
}

func parseSide(raw string) (types.Side, error) {
	switch strings.ToUpper(raw) {
	case string(types.SideBuy):
		return types.SideBuy, nil
	case string(types.SideSell):
		return types.SideSell, nil
	default:
		return "", fmt.Errorf("unsupported side %q", raw)
	}
}

func decimalFromString(field, value string) (decimal.Decimal, error) {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Decimal{}, status.Errorf(codes.InvalidArgument, "invalid %s: %q", field, value)
	}
	return d, nil
}

func toProtoTrades(trades []*types.Trade) []*pb.Trade {
	result := make([]*pb.Trade, 0, len(trades))
	for _, t := range trades {
		result = append(result, &pb.Trade{
			TradeId:     t.TradeID,
			BuyOrderId:  t.BuyOrderID,
			SellOrderId: t.SellOrderID,
			Price:       t.Price.String(),
			Quantity:    t.Quantity.String(),
			Timestamp:   t.Timestamp,
		})
	}
	return result
}

func toProtoLevels(levels []*domain.EngineOrderBookLevel) []*pb.OrderBookLevel {
	result := make([]*pb.OrderBookLevel, 0, len(levels))
	for _, level := range levels {
		result = append(result, &pb.OrderBookLevel{
			Price:    level.Price.String(),
			Quantity: level.Quantity.String(),
		})
	}
	return result
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	symbol := os.Getenv("MATCHING_SYMBOL")
	if symbol == "" {
		symbol = "BTC-USDT"
	}

	engine, err := domain.NewMatchingEngine(symbol, 1024*1024, logger)
	if err != nil {
		log.Fatalf("failed to initialize matching engine: %v", err)
	}
	if err := engine.Start(); err != nil {
		log.Fatalf("failed to start matching engine: %v", err)
	}
	defer engine.Shutdown()

	addr := os.Getenv("MATCHING_GRPC_ADDR")
	if addr == "" {
		addr = ":9094"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMatchingEngineServiceServer(s, newMatchingServer(engine))

	go func() {
		logger.Info("matching engine server started", "addr", addr, "symbol", symbol)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down matching engine server")
	s.GracefulStop()
}
