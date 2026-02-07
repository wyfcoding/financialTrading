package http

import (
	"net/http"
	"strconv"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/pkg/logging"
)

// MatchingHandler 负责处理 HTTP 请求
type MatchingHandler struct {
	cmd   *application.MatchingCommandService
	query *application.MatchingQueryService
}

func NewMatchingHandler(cmd *application.MatchingCommandService, query *application.MatchingQueryService) *MatchingHandler {
	return &MatchingHandler{cmd: cmd, query: query}
}

func (h *MatchingHandler) RegisterRoutes(router *gin.RouterGroup) {
	api := router.Group("/api/v1/matching")
	{
		api.POST("/orders", h.SubmitOrder)
		api.GET("/orderbook", h.GetOrderBook)
		api.GET("/trades", h.GetTrades)
	}
}

// SubmitOrder 处理订单提交请求并压入撮合引擎定序队列。
func (h *MatchingHandler) SubmitOrder(c *gin.Context) {
	var req application.SubmitOrderCommand
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid request data", err.Error())
		return
	}

	result, err := h.cmd.SubmitOrder(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "failed to submit order", "error", err)
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

// GetOrderBook 获取当前内存订单簿的快照（包含买卖盘档位）。
func (h *MatchingHandler) GetOrderBook(c *gin.Context) {
	depthStr := c.DefaultQuery("depth", "20")
	depth, err := strconv.Atoi(depthStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid depth parameter", "")
		return
	}

	snapshot, err := h.query.GetOrderBook(c.Request.Context(), depth)
	if err != nil {
		logging.Error(c.Request.Context(), "failed to get order book snapshot", "error", err)
		response.Error(c, err)
		return
	}

	response.Success(c, snapshot)
}

// GetTrades 获取指定交易对的最近成交历史。
func (h *MatchingHandler) GetTrades(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "symbol parameter is required", "")
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit", "")
		return
	}

	trades, err := h.query.GetTrades(c.Request.Context(), symbol, limit)
	if err != nil {
		logging.Error(c.Request.Context(), "failed to get trade history", "symbol", symbol, "error", err)
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"data": trades})
}
