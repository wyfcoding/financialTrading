package http

import (
	"net/http"
	"time"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/quant/application"
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
func (h *QuantHandler) RegisterRoutes(router *gin.RouterGroup) {
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
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	cmd := application.CreateStrategyCommand{
		Name:        req.Name,
		Description: req.Description,
		Script:      req.Script,
	}

	strategy, err := h.app.CreateStrategy(c.Request.Context(), cmd)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to create strategy", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{"strategy_id": strategy.ID})
}

// GetStrategy 获取策略
func (h *QuantHandler) GetStrategy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "id is required", "")
		return
	}

	strategy, err := h.app.GetStrategy(c.Request.Context(), id)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get strategy", "id", id, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	if strategy == nil {
		response.ErrorWithStatus(c, http.StatusNotFound, "strategy not found", "")
		return
	}

	response.Success(c, strategy)
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
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	cmd := application.RunBacktestCommand{
		StrategyID: req.StrategyID,
		Symbol:     req.Symbol,
		StartTime:  req.StartTime.UnixMilli(),
		EndTime:    req.EndTime.UnixMilli(),
	}

	result, err := h.app.RunBacktest(c.Request.Context(), cmd)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to run backtest", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{"backtest_id": result.ID})
}

// GetBacktestResult 获取回测结果
func (h *QuantHandler) GetBacktestResult(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "id is required", "")
		return
	}

	result, err := h.app.GetBacktestResult(c.Request.Context(), id)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get backtest result", "id", id, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	if result == nil {
		response.ErrorWithStatus(c, http.StatusNotFound, "result not found", "")
		return
	}

	response.Success(c, result)
}
