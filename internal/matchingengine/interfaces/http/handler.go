package http

import (
	"net/http"
	"strconv"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/matchingengine/application"
	"github.com/wyfcoding/pkg/logging"
)

// MatchingHandler 负责处理与撮合引擎相关的 HTTP 请求
type MatchingHandler struct {
	matchingService *application.MatchingEngineService
}

func NewMatchingHandler(matchingService *application.MatchingEngineService) *MatchingHandler {
	return &MatchingHandler{
		matchingService: matchingService,
	}
}

func (h *MatchingHandler) RegisterRoutes(router *gin.RouterGroup) {
	api := router.Group("/api/v1/matching")
	{
		api.POST("/orders", h.SubmitOrder)
		api.GET("/orderbook", h.GetOrderBook)
		api.GET("/trades", h.GetTrades)
	}
}

func (h *MatchingHandler) SubmitOrder(c *gin.Context) {
	var req application.SubmitOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	result, err := h.matchingService.SubmitOrder(c.Request.Context(), &req)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to submit order", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, result)
}

func (h *MatchingHandler) GetOrderBook(c *gin.Context) {
	depthStr := c.DefaultQuery("depth", "20")
	depth, err := strconv.Atoi(depthStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid depth", "")
		return
	}

	// 修正：GetOrderBook 不再需要 symbol
	snapshot, err := h.matchingService.GetOrderBook(c.Request.Context(), depth)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get order book", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, snapshot)
}

func (h *MatchingHandler) GetTrades(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "symbol is required", "")
		return
	}

	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit", "")
		return
	}

	trades, err := h.matchingService.GetTrades(c.Request.Context(), symbol, limit)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get trades", "symbol", symbol, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{"data": trades})
}
