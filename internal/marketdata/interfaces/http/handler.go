package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/marketdata/application"
)

type MarketDataHandler struct {
	query *application.MarketDataQueryService
}

func NewMarketDataHandler(query *application.MarketDataQueryService) *MarketDataHandler {
	return &MarketDataHandler{query: query}
}

func (h *MarketDataHandler) RegisterRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1/marketdata")
	{
		v1.GET("/quote", h.GetLatestQuote)
		v1.GET("/klines", h.GetKlines)
		v1.GET("/trades", h.GetTrades)
		v1.GET("/orderbook", h.GetOrderBook)
		v1.GET("/volatility", h.GetVolatility)
	}
}

func (h *MarketDataHandler) GetLatestQuote(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	dto, err := h.query.GetLatestQuote(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if dto == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "quote not found"})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *MarketDataHandler) GetKlines(c *gin.Context) {
	symbol := c.Query("symbol")
	interval := c.Query("interval")
	limitStr := c.Query("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	dtos, err := h.query.GetKlines(c.Request.Context(), symbol, interval, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "interval": interval, "klines": dtos})
}

func (h *MarketDataHandler) GetTrades(c *gin.Context) {
	symbol := c.Query("symbol")
	limitStr := c.Query("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	dtos, err := h.query.GetTrades(c.Request.Context(), symbol, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "trades": dtos})
}

func (h *MarketDataHandler) GetOrderBook(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	dto, err := h.query.GetOrderBook(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if dto == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "orderbook not found"})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *MarketDataHandler) GetVolatility(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	vol, err := h.query.GetVolatility(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"symbol": symbol, "volatility": vol.InexactFloat64()})
}
