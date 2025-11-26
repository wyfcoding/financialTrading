package http

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/reference-data/application"
	"github.com/wyfcoding/financialTrading/pkg/logger"
)

// ReferenceDataHandler HTTP 处理器
type ReferenceDataHandler struct {
	app *application.ReferenceDataService
}

// NewReferenceDataHandler 创建 HTTP 处理器实例
func NewReferenceDataHandler(app *application.ReferenceDataService) *ReferenceDataHandler {
	return &ReferenceDataHandler{app: app}
}

// RegisterRoutes 注册路由
func (h *ReferenceDataHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/reference-data")
	{
		api.GET("/symbols", h.ListSymbols)
		api.GET("/symbols/:id", h.GetSymbol)
		api.GET("/exchanges", h.ListExchanges)
		api.GET("/exchanges/:id", h.GetExchange)
	}
}

// GetSymbol 获取交易对
func (h *ReferenceDataHandler) GetSymbol(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	symbol, err := h.app.GetSymbol(c.Request.Context(), id)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to get symbol", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if symbol == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "symbol not found"})
		return
	}

	c.JSON(http.StatusOK, symbol)
}

// ListSymbols 列出交易对
func (h *ReferenceDataHandler) ListSymbols(c *gin.Context) {
	exchangeID := c.Query("exchange_id")
	status := c.Query("status")

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	symbols, err := h.app.ListSymbols(c.Request.Context(), exchangeID, status, limit, offset)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to list symbols", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, symbols)
}

// GetExchange 获取交易所
func (h *ReferenceDataHandler) GetExchange(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	exchange, err := h.app.GetExchange(c.Request.Context(), id)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to get exchange", "id", id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if exchange == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exchange not found"})
		return
	}

	c.JSON(http.StatusOK, exchange)
}

// ListExchanges 列出交易所
func (h *ReferenceDataHandler) ListExchanges(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}

	exchanges, err := h.app.ListExchanges(c.Request.Context(), limit, offset)
	if err != nil {
		logger.WithContext(c.Request.Context()).Error("Failed to list exchanges", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, exchanges)
}
