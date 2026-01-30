package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	clearingv1 "github.com/wyfcoding/financialtrading/go-api/clearing/v1"
	"github.com/wyfcoding/financialtrading/internal/clearing/application"
)

type ClearingHandler struct {
	app *application.ClearingService
}

func NewClearingHandler(app *application.ClearingService) *ClearingHandler {
	return &ClearingHandler{
		app: app,
	}
}

func (h *ClearingHandler) RegisterRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1/clearing")
	{
		v1.POST("/settle", h.SettleTrade)
		v1.GET("/settlement/:id", h.GetSettlement)
	}
}

func (h *ClearingHandler) SettleTrade(c *gin.Context) {
	var req clearingv1.SettleTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	qty, _ := decimal.NewFromString(req.Quantity)
	price, _ := decimal.NewFromString(req.Price)

	cmd := application.SettleTradeCommand{
		TradeID:    req.TradeId,
		BuyUserID:  req.BuyUserId,
		SellUserID: req.SellUserId,
		Symbol:     req.Symbol,
		Quantity:   qty,
		Price:      price,
	}

	dto, err := h.app.Command.SettleTrade(c.Request.Context(), &cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto)
}

func (h *ClearingHandler) GetSettlement(c *gin.Context) {
	id := c.Param("id")
	dto, err := h.app.Query.GetSettlement(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}
