package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与市场数据相关的 HTTP 请求
type Handler struct {
	service *application.MarketDataService // 市场数据应用服务
}

// 创建 HTTP 处理器实例
// service: 注入的市场数据应用服务
func NewHandler(service *application.MarketDataService) *Handler {
	return &Handler{
		service: service,
	}
}

// GetLatestQuote 获取最新行情
// @Summary 获取最新行情
// @Description 获取指定交易对的最新行情数据
// @Tags Market Data
// @Param symbol query string true "交易对符号"
// @Success 200 {object} QuoteResponse
// @Failure 400 {object} ErrorResponse
// @Router /api/v1/marketdata/quote [get]
func (h *Handler) GetLatestQuote(c *gin.Context) {
	symbol := c.Query("symbol")
	ctx := c.Request.Context()

	// 验证输入
	if symbol == "" {
		logging.Warn(ctx, "Invalid request: symbol is required")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol is required",
		})
		return
	}

	// 调用应用服务
	req := &application.GetLatestQuoteRequest{
		Symbol: symbol,
	}

	quoteDTO, err := h.service.GetLatestQuote(ctx, req)
	if err != nil {
		logging.Error(ctx, "Failed to get latest quote",
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
// @Router /api/v1/marketdata/quotes [get]
func (h *Handler) GetHistoricalQuotes(c *gin.Context) {
	symbol := c.Query("symbol")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")
	ctx := c.Request.Context()

	// 验证输入
	if symbol == "" || startTime == "" || endTime == "" {
		logging.Warn(ctx, "Invalid request parameters")
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol, start_time, and end_time are required",
		})
		return
	}

	// 解析 startTime 和 endTime 从字符串到 int64
	startTimeInt, err := strconv.ParseInt(startTime, 10, 64)
	if err != nil {
		logging.Warn(ctx, "Invalid start_time format",
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
		logging.Warn(ctx, "Invalid end_time format",
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
		logging.Warn(ctx, "Invalid time range",
			"start_time", startTimeInt,
			"end_time", endTimeInt,
		)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "start_time must be before end_time",
		})
		return
	}

	// 调用应用服务
	quotes, err := h.service.GetHistoricalQuotes(ctx, symbol, startTimeInt, endTimeInt)
	if err != nil {
		logging.Error(ctx, "Failed to get historical quotes",
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

// 注册路由
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	v1 := router.Group("/api/v1/marketdata")
	{
		v1.GET("/quote", h.GetLatestQuote)
		v1.GET("/quotes", h.GetHistoricalQuotes)
	}
}
