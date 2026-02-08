package http

import (
	"net/http"
	"strconv"
	"time"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/quant/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与量化策略和回测相关的 HTTP 请求
type QuantHandler struct {
	command *application.QuantCommandService
	query   *application.QuantQueryService
}

// 创建 HTTP 处理器实例
// app: 注入的量化应用服务
func NewQuantHandler(command *application.QuantCommandService, query *application.QuantQueryService) *QuantHandler {
	return &QuantHandler{command: command, query: query}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *QuantHandler) RegisterRoutes(router *gin.RouterGroup) {
	api := router.Group("/v1/quant")
	{
		api.POST("/strategies", h.CreateStrategy)
		api.GET("/strategies/:id", h.GetStrategy)
		api.POST("/backtests", h.RunBacktest)
		api.GET("/backtests/:id", h.GetBacktestResult)
		api.POST("/signals", h.GenerateSignal)
		api.GET("/signals", h.GetSignal)
		api.POST("/arbitrage/opportunities", h.FindArbitrageOpportunities)
		api.POST("/portfolio/optimize", h.OptimizePortfolio)
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

	strategy, err := h.command.CreateStrategy(c.Request.Context(), cmd)
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

	strategy, err := h.query.GetStrategy(c.Request.Context(), id)
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

	result, err := h.command.RunBacktest(c.Request.Context(), cmd)
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

	result, err := h.query.GetBacktestResult(c.Request.Context(), id)
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

// GenerateSignalRequest 生成信号请求
type GenerateSignalRequest struct {
	StrategyID string  `json:"strategy_id"`
	Symbol     string  `json:"symbol" binding:"required"`
	Indicator  string  `json:"indicator"`
	Period     int     `json:"period"`
	Value      float64 `json:"value"`
	Confidence float64 `json:"confidence"`
}

// GenerateSignal 生成信号
func (h *QuantHandler) GenerateSignal(c *gin.Context) {
	var req GenerateSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	cmd := application.GenerateSignalCommand{
		StrategyID: req.StrategyID,
		Symbol:     req.Symbol,
		Indicator:  req.Indicator,
		Period:     req.Period,
		Value:      req.Value,
		Confidence: req.Confidence,
	}

	signal, err := h.command.GenerateSignal(c.Request.Context(), cmd)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to generate signal", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, signal)
}

// GetSignal 获取信号
func (h *QuantHandler) GetSignal(c *gin.Context) {
	symbol := c.Query("symbol")
	indicator := c.Query("indicator")
	periodStr := c.Query("period")
	if symbol == "" || indicator == "" || periodStr == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "symbol/indicator/period are required", "")
		return
	}
	period, err := strconv.Atoi(periodStr)
	if err != nil || period <= 0 {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid period", "")
		return
	}
	dto, err := h.query.GetSignal(c.Request.Context(), symbol, indicator, period)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get signal", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}
	response.Success(c, dto)
}

// OptimizePortfolioRequest 优化组合请求
type OptimizePortfolioRequest struct {
	PortfolioID    string   `json:"portfolio_id"`
	Symbols        []string `json:"symbols" binding:"required"`
	ExpectedReturn float64  `json:"expected_return"`
	RiskTolerance  float64  `json:"risk_tolerance"`
}

// OptimizePortfolio 优化组合
func (h *QuantHandler) OptimizePortfolio(c *gin.Context) {
	var req OptimizePortfolioRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	cmd := application.OptimizePortfolioCommand{
		PortfolioID:    req.PortfolioID,
		Symbols:        req.Symbols,
		ExpectedReturn: req.ExpectedReturn,
		RiskTolerance:  req.RiskTolerance,
	}

	weights, err := h.command.OptimizePortfolio(c.Request.Context(), cmd)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to optimize portfolio", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{
		"portfolio_id": req.PortfolioID,
		"weights":      weights,
	})
}

// FindArbitrageRequest 查找套利机会
type FindArbitrageRequest struct {
	Symbol string   `json:"symbol" binding:"required"`
	Venues []string `json:"venues" binding:"required"`
}

// FindArbitrageOpportunities 查找套利机会
func (h *QuantHandler) FindArbitrageOpportunities(c *gin.Context) {
	var req FindArbitrageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}
	opps, err := h.query.FindArbitrageOpportunities(c.Request.Context(), req.Symbol, req.Venues)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to find arbitrage opportunities", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}
	response.Success(c, gin.H{"opportunities": opps})
}
