package http

import (
	"net/http"
	"strconv"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/referencedata/application"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与参考数据相关的 HTTP 请求
type ReferenceDataHandler struct {
	app *application.ReferenceDataService // 参考数据应用服务
}

// 创建 HTTP 处理器实例
// app: 注入的参考数据应用服务
func NewReferenceDataHandler(app *application.ReferenceDataService) *ReferenceDataHandler {
	return &ReferenceDataHandler{app: app}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *ReferenceDataHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/referencedata")
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
		response.ErrorWithStatus(c, http.StatusBadRequest, "id is required", "")
		return
	}

	symbol, err := h.app.GetSymbol(c.Request.Context(), id)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get symbol", "id", id, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	if symbol == nil {
		response.ErrorWithStatus(c, http.StatusNotFound, "symbol not found", "")
		return
	}

	response.Success(c, symbol)
}

// ListSymbols 列出交易对
func (h *ReferenceDataHandler) ListSymbols(c *gin.Context) {
	exchangeID := c.Query("exchange_id")
	status := c.Query("status")

	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit", "")
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid offset", "")
		return
	}

	symbols, err := h.app.ListSymbols(c.Request.Context(), exchangeID, status, limit, offset)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to list symbols", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, symbols)
}

// GetExchange 获取交易所
func (h *ReferenceDataHandler) GetExchange(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		response.ErrorWithStatus(c, http.StatusBadRequest, "id is required", "")
		return
	}

	exchange, err := h.app.GetExchange(c.Request.Context(), id)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to get exchange", "id", id, "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	if exchange == nil {
		response.ErrorWithStatus(c, http.StatusNotFound, "exchange not found", "")
		return
	}

	response.Success(c, exchange)
}

// ListExchanges 列出交易所
func (h *ReferenceDataHandler) ListExchanges(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid limit", "")
		return
	}

	offsetStr := c.DefaultQuery("offset", "0")
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, "invalid offset", "")
		return
	}

	exchanges, err := h.app.ListExchanges(c.Request.Context(), limit, offset)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to list exchanges", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, exchanges)
}
