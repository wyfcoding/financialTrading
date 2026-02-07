package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wyfcoding/financialtrading/internal/order/domain"
	"github.com/wyfcoding/pkg/contextx"
)

// OrderCommandService 处理订单相关的命令操作
// 使用事件溯源 + Outbox

type OrderCommandService struct {
	repo           domain.OrderRepository
	eventStore     domain.EventStore
	eventPublisher domain.EventPublisher
}

// NewOrderCommandService 创建新的 OrderCommandService 实例
func NewOrderCommandService(repo domain.OrderRepository, eventStore domain.EventStore, eventPublisher domain.EventPublisher) *OrderCommandService {
	return &OrderCommandService{
		repo:           repo,
		eventStore:     eventStore,
		eventPublisher: eventPublisher,
	}
}

// PlaceOrder 下单
func (c *OrderCommandService) PlaceOrder(ctx context.Context, cmd PlaceOrderCommand) (string, error) {
	if err := validatePlaceOrder(cmd); err != nil {
		return "", err
	}

	orderID := generateOrderID()
	var createdID string

	err := c.repo.WithTx(ctx, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)

		order := domain.NewOrder(
			orderID,
			cmd.UserID,
			cmd.Symbol,
			domain.OrderSide(cmd.Side),
			domain.OrderType(cmd.Type),
			cmd.Price,
			cmd.Quantity,
			cmd.StopPrice,
			domain.TimeInForce(cmd.TimeInForce),
			cmd.ParentOrderID,
			cmd.IsOCO,
		)

		if err := order.Validate(); err != nil {
			return err
		}

		if err := c.repo.Save(txCtx, order); err != nil {
			return err
		}

		if err := c.eventStore.Save(txCtx, order.OrderID, order.GetUncommittedEvents(), order.Version()); err != nil {
			return err
		}
		if c.eventPublisher != nil {
			for _, ev := range order.GetUncommittedEvents() {
				if err := c.eventPublisher.PublishInTx(txCtx, tx, ev.EventType(), order.OrderID, ev); err != nil {
					return err
				}
			}
		}
		order.MarkCommitted()

		order.MarkValidated()
		if err := c.repo.Save(txCtx, order); err != nil {
			return err
		}
		if err := c.eventStore.Save(txCtx, order.OrderID, order.GetUncommittedEvents(), order.Version()); err != nil {
			return err
		}
		if c.eventPublisher != nil {
			for _, ev := range order.GetUncommittedEvents() {
				if err := c.eventPublisher.PublishInTx(txCtx, tx, ev.EventType(), order.OrderID, ev); err != nil {
					return err
				}
			}
		}
		order.MarkCommitted()

		createdID = order.OrderID
		return nil
	})

	if err != nil {
		return "", err
	}
	return createdID, nil
}

// CancelOrder 取消订单
func (c *OrderCommandService) CancelOrder(ctx context.Context, cmd CancelOrderCommand) error {
	if cmd.OrderID == "" {
		return errors.New("order_id is required")
	}
	return c.repo.WithTx(ctx, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)

		order, err := c.repo.Get(txCtx, cmd.OrderID)
		if err != nil {
			return err
		}
		if order == nil {
			return fmt.Errorf("order not found")
		}
		if cmd.UserID != "" && order.UserID != cmd.UserID {
			return ErrUnauthorized
		}
		if order.Status != domain.StatusValidated && order.Status != domain.StatusPartiallyFilled {
			return ErrInvalidOrderStatus
		}

		order.ApplyChange(&domain.OrderCancelledEvent{
			OrderID:     order.OrderID,
			UserID:      order.UserID,
			Symbol:      order.Symbol,
			Reason:      cmd.Reason,
			CancelledAt: time.Now().UnixNano(),
			OccurredOn:  time.Now(),
		})

		if err := c.repo.Save(txCtx, order); err != nil {
			return err
		}
		if err := c.eventStore.Save(txCtx, order.OrderID, order.GetUncommittedEvents(), order.Version()); err != nil {
			return err
		}
		if c.eventPublisher != nil {
			for _, ev := range order.GetUncommittedEvents() {
				if err := c.eventPublisher.PublishInTx(txCtx, tx, ev.EventType(), order.OrderID, ev); err != nil {
					return err
				}
			}
		}
		order.MarkCommitted()
		return nil
	})
}

// UpdateOrderExecution 更新订单执行状态
func (c *OrderCommandService) UpdateOrderExecution(ctx context.Context, orderID string, filledQty, tradePrice float64) error {
	if orderID == "" {
		return errors.New("order_id is required")
	}
	return c.repo.WithTx(ctx, func(txCtx context.Context) error {
		tx := contextx.GetTx(txCtx)
		order, err := c.repo.Get(txCtx, orderID)
		if err != nil {
			return err
		}
		if order == nil {
			return fmt.Errorf("order not found")
		}

		order.UpdateExecution(filledQty, tradePrice)
		if err := c.repo.Save(txCtx, order); err != nil {
			return err
		}
		if err := c.eventStore.Save(txCtx, order.OrderID, order.GetUncommittedEvents(), order.Version()); err != nil {
			return err
		}
		if c.eventPublisher != nil {
			for _, ev := range order.GetUncommittedEvents() {
				if err := c.eventPublisher.PublishInTx(txCtx, tx, ev.EventType(), order.OrderID, ev); err != nil {
					return err
				}
			}
		}
		order.MarkCommitted()
		return nil
	})
}

// 生成订单 ID
func generateOrderID() string {
	return "ORDER_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// 生成随机字符串
func randomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[time.Now().UnixNano()%int64(len(letterBytes))]
	}
	return string(b)
}

// 错误定义
var (
	ErrUnauthorized       = NewError("unauthorized", "unauthorized to cancel this order")
	ErrInvalidOrderStatus = NewError("invalid_order_status", "order status cannot be cancelled")
)

// Error 自定义错误

type Error struct {
	Code    string
	Message string
}

// Error 实现 error 接口
func (e *Error) Error() string {
	return e.Message
}

// NewError 创建新的错误
func NewError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

func validatePlaceOrder(cmd PlaceOrderCommand) error {
	if cmd.UserID == "" || cmd.Symbol == "" || cmd.Side == "" || cmd.Type == "" {
		return errors.New("user_id, symbol, side, type are required")
	}
	cmd.Side = strings.ToLower(cmd.Side)
	if cmd.Side != string(domain.SideBuy) && cmd.Side != string(domain.SideSell) {
		return errors.New("invalid side")
	}
	cmd.Type = strings.ToLower(cmd.Type)
	if cmd.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	if cmd.Type == string(domain.TypeLimit) && cmd.Price <= 0 {
		return errors.New("price must be positive for limit orders")
	}
	return nil
}
