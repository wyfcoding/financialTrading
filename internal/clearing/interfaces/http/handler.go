package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/clearing/application"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// ClearingHandler HTTP 处理器
// 负责处理与清算相关的 HTTP 请求
type ClearingHandler struct {
	clearingService *application.ClearingApplicationService // 清算应用服务
}

// NewClearingHandler 创建 HTTP 处理器实例
// clearingService: 注入的清算应用服务
func NewClearingHandler(clearingService *application.ClearingApplicationService) *ClearingHandler {
	return &ClearingHandler{
		clearingService: clearingService,
	}
}

// RegisterRoutes 注册路由
func (h *ClearingHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/clearing")
	{
		api.POST("/settle", h.SettleTrade)
		api.POST("/eod", h.ExecuteEODClearing)
		api.GET("/:id", h.GetClearingStatus)
	}
}

// SettleTrade 清算交易
func (h *ClearingHandler) SettleTrade(c *gin.Context) {
	var req application.SettleTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clearingService.SettleTrade(c.Request.Context(), &req); err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to settle trade", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// ExecuteEODClearingRequest 执行日终清算请求
type ExecuteEODClearingRequest struct {
	ClearingDate string `json:"clearing_date" binding:"required"`
}

// ExecuteEODClearing 执行日终清算
func (h *ClearingHandler) ExecuteEODClearing(c *gin.Context) {
	var req ExecuteEODClearingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.clearingService.ExecuteEODClearing(c.Request.Context(), req.ClearingDate); err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to execute EOD clearing", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "processing"})
}

// GetClearingStatus 获取清算状态
func (h *ClearingHandler) GetClearingStatus(c *gin.Context) {
	clearingID := c.Param("id")
	if clearingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clearing_id is required"})
		return
	}

	clearing, err := h.clearingService.GetClearingStatus(c.Request.Context(), clearingID)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to get clearing status", "clearing_id", clearingID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if clearing == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "clearing not found"})
		return
	}

	c.JSON(http.StatusOK, clearing)
}
