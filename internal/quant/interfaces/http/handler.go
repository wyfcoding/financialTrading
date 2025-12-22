package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/quant/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与量化策略和回测相关的 HTTP 请求
type QuantHandler struct {
	app *application.QuantService // 量化应用服务
}

// 创建 HTTP 处理器实例
// app: 注入的量化应用服务
func NewQuantHandler(app *application.QuantService) *QuantHandler {
	return &QuantHandler{app: app}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *QuantHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/quant")
	{
		api.POST("/strategies", h.CreateStrategy)
		api.GET("/strategies/:id", h.GetStrategy)
		api.POST("/backtests", h.RunBacktest)
		api.GET("/backtests/:id", h.GetBacktestResult)
	}
}

// CreateStrategyRequest 创建策略请求
type CreateStrategyRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Script      string `json:"script" binding:"required"`
}

// CreateStrategy 创建策略
func (h *QuantHandler) CreateStrategy(c *gin.Context) {
	var req CreateStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.app.CreateStrategy(c.Request.Context(), req.Name, req.Description, req.Script)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to create strategy", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"strategy_id": id})
}

// GetStrategy 获取策略
func (h *QuantHandler) GetStrategy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	strategy, err := h.app.GetStrategy(c.Request.Context(), id)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get strategy", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if strategy == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "strategy not found"})
		return
	}

	c.JSON(http.StatusOK, strategy)
}

// RunBacktestRequest 运行回测请求
type RunBacktestRequest struct {
	StrategyID     string    `json:"strategy_id" binding:"required"`
	Symbol         string    `json:"symbol" binding:"required"`
	StartTime      time.Time `json:"start_time" binding:"required"`
	EndTime        time.Time `json:"end_time" binding:"required"`
	InitialCapital float64   `json:"initial_capital" binding:"required"`
}

// RunBacktest 运行回测
func (h *QuantHandler) RunBacktest(c *gin.Context) {
	var req RunBacktestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.app.RunBacktest(c.Request.Context(), req.StrategyID, req.Symbol, req.StartTime, req.EndTime, req.InitialCapital)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to run backtest", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"backtest_id": id})
}

// GetBacktestResult 获取回测结果
func (h *QuantHandler) GetBacktestResult(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	result, err := h.app.GetBacktestResult(c.Request.Context(), id)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get backtest result", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "result not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}
