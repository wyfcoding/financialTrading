// Package interfaces 算法交易接口层
package interfaces

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/algotrading/application"
	"github.com/wyfcoding/financialtrading/internal/algotrading/domain"
)

// HTTPHandler HTTP 接口处理器
type HTTPHandler struct {
	commandService *application.CommandService
	queryService   *application.QueryService
}

// NewHTTPHandler 创建 HTTP 处理器
func NewHTTPHandler(
	commandService *application.CommandService,
	queryService *application.QueryService,
) *HTTPHandler {
	return &HTTPHandler{
		commandService: commandService,
		queryService:   queryService,
	}
}

// RegisterRoutes 注册路由
func (h *HTTPHandler) RegisterRoutes(r *gin.RouterGroup) {
	algo := r.Group("/algotrading")
	{
		algo.POST("/strategies", h.CreateStrategy)
		algo.POST("/strategies/:id/start", h.StartStrategy)
		algo.POST("/strategies/:id/stop", h.StopStrategy)
		algo.GET("/strategies/:id", h.GetStrategy)
		algo.POST("/backtests", h.SubmitBacktest)
		algo.GET("/backtests/:id", h.GetBacktestResult)
	}
}

// CreateStrategyRequest 创建策略请求
type CreateStrategyRequest struct {
	UserID     uint64 `json:"user_id" binding:"required"`
	Type       int8   `json:"type" binding:"required"`
	Symbol     string `json:"symbol" binding:"required"`
	Parameters string `json:"parameters" binding:"required"`
}

// CreateStrategy 创建策略
func (h *HTTPHandler) CreateStrategy(c *gin.Context) {
	var req CreateStrategyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := application.CreateStrategyCommand{
		UserID:     req.UserID,
		Type:       domain.StrategyType(req.Type),
		Symbol:     req.Symbol,
		Parameters: req.Parameters,
	}

	id, err := h.commandService.CreateStrategy(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"strategy_id": id})
}

// StartStrategy 启动策略
func (h *HTTPHandler) StartStrategy(c *gin.Context) {
	id := c.Param("id")
	if err := h.commandService.StartStrategy(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "started"})
}

// StopStrategy 停止策略
func (h *HTTPHandler) StopStrategy(c *gin.Context) {
	id := c.Param("id")
	if err := h.commandService.StopStrategy(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stopped"})
}

// SubmitBacktestRequest 提交回测请求
type SubmitBacktestRequest struct {
	UserID     uint64 `json:"user_id" binding:"required"`
	Type       int8   `json:"type" binding:"required"`
	Symbol     string `json:"symbol" binding:"required"`
	Parameters string `json:"parameters" binding:"required"`
	StartTime  string `json:"start_time" binding:"required"`
	EndTime    string `json:"end_time" binding:"required"`
}

// SubmitBacktest 提交回测
func (h *HTTPHandler) SubmitBacktest(c *gin.Context) {
	var req SubmitBacktestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format"})
		return
	}
	end, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format"})
		return
	}

	cmd := application.SubmitBacktestCommand{
		UserID:     req.UserID,
		Type:       domain.StrategyType(req.Type),
		Symbol:     req.Symbol,
		Parameters: req.Parameters,
		StartTime:  start,
		EndTime:    end,
	}

	id, err := h.commandService.SubmitBacktest(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"backtest_id": id})
}

// GetStrategy 获取策略
func (h *HTTPHandler) GetStrategy(c *gin.Context) {
	id := c.Param("id")
	strategy, err := h.queryService.GetStrategy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, strategy)
}

// GetBacktestResult 获取回测结果
func (h *HTTPHandler) GetBacktestResult(c *gin.Context) {
	id := c.Param("id")
	result, err := h.queryService.GetBacktestResult(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}
