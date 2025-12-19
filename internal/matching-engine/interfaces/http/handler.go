package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/matching-engine/application"
	"github.com/wyfcoding/pkg/logging"
)

// MatchingHandler HTTP 处理器
// 负责处理与撮合引擎相关的 HTTP 请求
type MatchingHandler struct {
	matchingService *application.MatchingApplicationService // 撮合应用服务
}

// NewMatchingHandler 创建 HTTP 处理器
// matchingService: 注入的撮合应用服务
func NewMatchingHandler(matchingService *application.MatchingApplicationService) *MatchingHandler {
	return &MatchingHandler{
		matchingService: matchingService,
	}
}

// RegisterRoutes 注册路由
func (h *MatchingHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/matching")
	{
		api.POST("/orders", h.SubmitOrder)
		api.GET("/orderbook", h.GetOrderBook)
		api.GET("/trades", h.GetTrades)
	}
}

// SubmitOrder 提交订单
func (h *MatchingHandler) SubmitOrder(c *gin.Context) {
	var req application.SubmitOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.matchingService.SubmitOrder(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to submit order", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetOrderBook 获取订单簿
func (h *MatchingHandler) GetOrderBook(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	depthStr := c.DefaultQuery("depth", "20")
	depth, err := strconv.Atoi(depthStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid depth"})
		return
	}

	snapshot, err := h.matchingService.GetOrderBook(c.Request.Context(), symbol, depth)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get order book", "symbol", symbol, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, snapshot)
}

// GetTrades 获取成交历史
func (h *MatchingHandler) GetTrades(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	trades, err := h.matchingService.GetTrades(c.Request.Context(), symbol, limit)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get trades", "symbol", symbol, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": trades})
}
