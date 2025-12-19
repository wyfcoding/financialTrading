package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/order/application"
	"github.com/wyfcoding/pkg/logging"
)

// OrderHandler HTTP 处理器
// 负责处理与订单相关的 HTTP 请求
type OrderHandler struct {
	orderService *application.OrderApplicationService // 订单应用服务
}

// NewOrderHandler 创建 HTTP 处理器实例
// orderService: 注入的订单应用服务
func NewOrderHandler(orderService *application.OrderApplicationService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
	}
}

// RegisterRoutes 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *OrderHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/orders")
	{
		api.POST("", h.CreateOrder)       // 创建订单
		api.DELETE("/:id", h.CancelOrder) // 取消订单
		api.GET("/:id", h.GetOrder)       // 获取订单详情
	}
}

// CreateOrder 创建订单
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req application.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dto, err := h.orderService.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to create order", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

// CancelOrder 取消订单
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
		return
	}

	// Assuming user_id is passed in header or context (e.g. from auth middleware)
	// For now, we'll try to get it from query param for simplicity if not in context
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	dto, err := h.orderService.CancelOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to cancel order", "order_id", orderID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

// GetOrder 获取订单
func (h *OrderHandler) GetOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
		return
	}

	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	dto, err := h.orderService.GetOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get order", "order_id", orderID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}
