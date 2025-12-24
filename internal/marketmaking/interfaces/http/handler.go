package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与做市相关的 HTTP 请求
type MarketMakingHandler struct {
	app *application.MarketMakingService // 做市应用服务
}

// 创建 HTTP 处理器实例
// app: 注入的做市应用服务
func NewMarketMakingHandler(app *application.MarketMakingService) *MarketMakingHandler {
	return &MarketMakingHandler{app: app}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *MarketMakingHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/marketmaking")
	{
		api.POST("/strategy", h.SetStrategy)
		api.GET("/strategy", h.GetStrategy)
		api.GET("/performance", h.GetPerformance)
	}
}

// SetStrategyRequest 设置策略请求
type SetStrategyRequest struct {
	Symbol       string  `json:"symbol" binding:"required"`
	Spread       float64 `json:"spread" binding:"required"`
	MinOrderSize float64 `json:"min_order_size" binding:"required"`
	MaxOrderSize float64 `json:"max_order_size" binding:"required"`
	MaxPosition  float64 `json:"max_position" binding:"required"`
	Status       string  `json:"status" binding:"required"`
}

// SetStrategy 设置做市策略
func (h *MarketMakingHandler) SetStrategy(c *gin.Context) {
	var req SetStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.app.SetStrategy(c.Request.Context(), req.Symbol, req.Spread, req.MinOrderSize, req.MaxOrderSize, req.MaxPosition, req.Status)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to set strategy", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"strategy_id": id})
}

// GetStrategy 获取做市策略
func (h *MarketMakingHandler) GetStrategy(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	strategy, err := h.app.GetStrategy(c.Request.Context(), symbol)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get strategy", "symbol", symbol, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if strategy == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "strategy not found"})
		return
	}

	c.JSON(http.StatusOK, strategy)
}

// GetPerformance 获取做市绩效
func (h *MarketMakingHandler) GetPerformance(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	performance, err := h.app.GetPerformance(c.Request.Context(), symbol)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get performance", "symbol", symbol, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if performance == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "performance not found"})
		return
	}

	c.JSON(http.StatusOK, performance)
}
