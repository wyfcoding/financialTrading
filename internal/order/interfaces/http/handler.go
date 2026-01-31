package http

import (
	"net/http"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/order/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与订单相关的 HTTP 请求
type OrderHandler struct {
	app *application.OrderService
}

// 创建 HTTP 处理器实例
func NewOrderHandler(app *application.OrderService) *OrderHandler {
	return &OrderHandler{
		app: app,
	}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *OrderHandler) RegisterRoutes(router *gin.RouterGroup) {
	api := router.Group("/api/v1/orders")
	{
		api.POST("", h.CreateOrder)       // 创建订单
		api.DELETE("/:id", h.CancelOrder) // 取消订单
		api.GET("/:id", h.GetOrder)       // 获取订单详情
	}
}

// CreateOrderRequest 创建订单请求

type CreateOrderRequest struct {
	UserID        string  `json:"user_id" binding:"required"`
	Symbol        string  `json:"symbol" binding:"required"`
	Side          string  `json:"side" binding:"required"`
	Type          string  `json:"type" binding:"required"`
	Price         float64 `json:"price" binding:"required"`
	StopPrice     float64 `json:"stop_price"`
	Quantity      float64 `json:"quantity" binding:"required"`
	TimeInForce   string  `json:"time_in_force"`
	ParentOrderID string  `json:"parent_order_id"`
	IsOCO         bool    `json:"is_oco"`
}

// CreateOrder 创建订单
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	cmd := application.PlaceOrderCommand{
		UserID:        req.UserID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Type:          req.Type,
		Price:         req.Price,
		StopPrice:     req.StopPrice,
		Quantity:      req.Quantity,
		TimeInForce:   req.TimeInForce,
		ParentOrderID: req.ParentOrderID,
		IsOCO:         req.IsOCO,
	}

	orderID, err := h.app.PlaceOrder(c.Request.Context(), cmd)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to create order", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{"order_id": orderID})
}

// CancelOrderRequest 取消订单请求

type CancelOrderRequest struct {
	UserID string `json:"user_id" binding:"required"`
	Reason string `json:"reason"`
}

// CancelOrder 取消订单
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "order_id is required", "")
		return
	}

	var req CancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	cmd := application.CancelOrderCommand{
		OrderID: orderID,
		UserID:  req.UserID,
		Reason:  req.Reason,
	}

	if err := h.app.CancelOrder(c.Request.Context(), cmd); err != nil {
		logging.Error(c.Request.Context(), "Failed to cancel order", "order_id", orderID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{"status": "cancelled", "order_id": orderID})
}

// GetOrder 获取订单
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "order_id is required", "")
		return
	}

	userID := c.Query("user_id")
	if userID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "user_id is required", "")
		return
	}

	dto, err := h.app.GetOrder(c.Request.Context(), orderID) // query typically doesn't need userID if it's broad
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get order", "order_id", orderID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dto)
}
