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
	orderService *application.OrderApplicationService // 订单应用服务
}

// 创建 HTTP 处理器实例
// orderService: 注入的订单应用服务
func NewOrderHandler(orderService *application.OrderApplicationService) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
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

// CreateOrder 创建订单
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	var req application.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	dto, err := h.orderService.CreateOrder(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to create order", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dto)
}

// CancelOrder 取消订单
func (h *OrderHandler) CancelOrder(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "order_id is required", "")
		return
	}

	// 假设 user_id 通过头部或上下文传递（例如来自认证中间件）
	// 如果上下文中没有，为简单起见，我们目前尝试从查询参数中获取
	userID := c.Query("user_id")
	if userID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "user_id is required", "")
		return
	}

	dto, err := h.orderService.CancelOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to cancel order", "order_id", orderID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dto)
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

	dto, err := h.orderService.GetOrder(c.Request.Context(), orderID, userID)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get order", "order_id", orderID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dto)
}
