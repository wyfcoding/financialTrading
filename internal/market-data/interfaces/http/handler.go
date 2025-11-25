// Package http 包含 HTTP 处理器、入参出参转换、校验与中间件接入
package http

import (
	"net/http"

	"github.com/fynnwu/FinancialTrading/internal/market-data/application"
	"github.com/fynnwu/FinancialTrading/pkg/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler HTTP 处理器
type Handler struct {
	quoteService *application.QuoteApplicationService
}

// NewHandler 创建 HTTP 处理器
func NewHandler(quoteService *application.QuoteApplicationService) *Handler {
	return &Handler{
		quoteService: quoteService,
	}
}

// GetLatestQuote 获取最新行情
// @Summary 获取最新行情
// @Description 获取指定交易对的最新行情数据
// @Tags Market Data
// @Param symbol query string true "交易对符号"
// @Success 200 {object} QuoteResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/market-data/quote [get]
func (h *Handler) GetLatestQuote(c *gin.Context) {
	symbol := c.Query("symbol")

	// 验证输入
	if symbol == "" {
		logger.WithFields(zap.String("error", "symbol is required")).Warn("Invalid request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol is required",
		})
		return
	}

	// 调用应用服务
	req := &application.GetLatestQuoteRequest{
		Symbol: symbol,
	}

	quoteDTO, err := h.quoteService.GetLatestQuote(c.Request.Context(), req)
	if err != nil {
		logger.WithFields(
			zap.String("symbol", symbol),
			zap.Error(err),
		).Error("Failed to get latest quote")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"data": quoteDTO,
	})
}

// GetHistoricalQuotes 获取历史行情
// @Summary 获取历史行情
// @Description 获取指定交易对的历史行情数据
// @Tags Market Data
// @Param symbol query string true "交易对符号"
// @Param start_time query int64 true "开始时间（毫秒）"
// @Param end_time query int64 true "结束时间（毫秒）"
// @Success 200 {object} HistoricalQuotesResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/market-data/quotes [get]
func (h *Handler) GetHistoricalQuotes(c *gin.Context) {
	symbol := c.Query("symbol")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	// 验证输入
	if symbol == "" || startTime == "" || endTime == "" {
		logger.Warn("Invalid request parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol, start_time, and end_time are required",
		})
		return
	}

	// 调用应用服务
	quotes, err := h.quoteService.GetHistoricalQuotes(c.Request.Context(), symbol, 0, 0)
	if err != nil {
		logger.WithFields(
			zap.String("symbol", symbol),
			zap.Error(err),
		).Error("Failed to get historical quotes")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{
		"data": quotes,
	})
}

// QuoteResponse 行情响应
type QuoteResponse struct {
	Symbol    string `json:"symbol"`
	BidPrice  string `json:"bid_price"`
	AskPrice  string `json:"ask_price"`
	BidSize   string `json:"bid_size"`
	AskSize   string `json:"ask_size"`
	LastPrice string `json:"last_price"`
	LastSize  string `json:"last_size"`
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Error string `json:"error"`
}

// HistoricalQuotesResponse 历史行情响应
type HistoricalQuotesResponse struct {
	Data []*QuoteResponse `json:"data"`
}

// RegisterRoutes 注册路由
func (h *Handler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1/market-data")
	{
		v1.GET("/quote", h.GetLatestQuote)
		v1.GET("/quotes", h.GetHistoricalQuotes)
	}
}
