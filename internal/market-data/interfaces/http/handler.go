package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/market-data/application"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// Handler HTTP 处理器
// 负责处理与市场数据相关的 HTTP 请求
type Handler struct {
	quoteService *application.QuoteApplicationService // 行情应用服务
}

// NewHandler 创建 HTTP 处理器实例
// quoteService: 注入的行情应用服务
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
	ctx := c.Request.Context()

	// 验证输入
	if symbol == "" {
		logger.Warn(ctx, "Invalid request: symbol is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol is required",
		})
		return
	}

	// 调用应用服务
	req := &application.GetLatestQuoteRequest{
		Symbol: symbol,
	}

	quoteDTO, err := h.quoteService.GetLatestQuote(ctx, req)
	if err != nil {
		logger.Error(ctx, "Failed to get latest quote",
			"symbol", symbol,
			"error", err,
		)
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
	ctx := c.Request.Context()

	// 验证输入
	if symbol == "" || startTime == "" || endTime == "" {
		logger.Warn(ctx, "Invalid request parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol, start_time, and end_time are required",
		})
		return
	}

	// 解析 startTime 和 endTime 从字符串到 int64
	startTimeInt, err := strconv.ParseInt(startTime, 10, 64)
	if err != nil {
		logger.Warn(ctx, "Invalid start_time format",
			"start_time", startTime,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "start_time must be a valid int64 timestamp",
		})
		return
	}

	endTimeInt, err := strconv.ParseInt(endTime, 10, 64)
	if err != nil {
		logger.Warn(ctx, "Invalid end_time format",
			"end_time", endTime,
			"error", err,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "end_time must be a valid int64 timestamp",
		})
		return
	}

	// 验证时间范围合法性
	if startTimeInt >= endTimeInt {
		logger.Warn(ctx, "Invalid time range",
			"start_time", startTimeInt,
			"end_time", endTimeInt,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "start_time must be before end_time",
		})
		return
	}

	// 调用应用服务
	quotes, err := h.quoteService.GetHistoricalQuotes(ctx, symbol, startTimeInt, endTimeInt)
	if err != nil {
		logger.Error(ctx, "Failed to get historical quotes",
			"symbol", symbol,
			"error", err,
		)
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
