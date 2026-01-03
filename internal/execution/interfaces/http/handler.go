package http

import (
	"net/http"
	"strconv"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/execution/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与订单执行相关的 HTTP 请求
type ExecutionHandler struct {
	executionService *application.ExecutionService // 执行应用服务
}

// 创建 HTTP 处理器实例
// executionService: 注入的执行应用服务
func NewExecutionHandler(executionService *application.ExecutionService) *ExecutionHandler {
	return &ExecutionHandler{
		executionService: executionService,
	}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *ExecutionHandler) RegisterRoutes(router *gin.RouterGroup) {
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
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	dto, err := h.executionService.ExecuteOrder(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to execute order", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, dto)
}

// GetExecutionHistory 获取执行历史
func (h *ExecutionHandler) GetExecutionHistory(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "user_id is required", "")
		return
	}

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit", "")
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid offset", "")
		return
	}

	dtos, total, err := h.executionService.GetExecutionHistory(c.Request.Context(), userID, limit, offset)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get execution history", "user_id", userID, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{
		"data":  dtos,
		"total": total,
	})
}
