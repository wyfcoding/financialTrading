package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/execution/application"
	"github.com/wyfcoding/pkg/logging"
)

// ExecutionHandler HTTP 处理器
// 负责处理与订单执行相关的 HTTP 请求
type ExecutionHandler struct {
	executionService *application.ExecutionApplicationService // 执行应用服务
}

// NewExecutionHandler 创建 HTTP 处理器实例
// executionService: 注入的执行应用服务
func NewExecutionHandler(executionService *application.ExecutionApplicationService) *ExecutionHandler {
	return &ExecutionHandler{
		executionService: executionService,
	}
}

// RegisterRoutes 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *ExecutionHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/execution")
	{
		api.POST("/orders", h.ExecuteOrder)        // 执行订单
		api.GET("/history", h.GetExecutionHistory) // 获取执行历史
	}
}

// ExecuteOrder 执行订单
func (h *ExecutionHandler) ExecuteOrder(c *gin.Context) {
	var req application.ExecuteOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dto, err := h.executionService.ExecuteOrder(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to execute order", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

// GetExecutionHistory 获取执行历史
func (h *ExecutionHandler) GetExecutionHistory(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	dtos, total, err := h.executionService.GetExecutionHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get execution history", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  dtos,
		"total": total,
	})
}
