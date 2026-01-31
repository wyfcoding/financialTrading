package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialtrading/internal/marketmaking/application"
)

type MarketMakingHandler struct {
	app *application.MarketMakingService
}

func NewMarketMakingHandler(app *application.MarketMakingService) *MarketMakingHandler {
	return &MarketMakingHandler{app: app}
}

func (h *MarketMakingHandler) RegisterRoutes(r *gin.RouterGroup) {
	v1 := r.Group("/v1/marketmaking")
	{
		v1.POST("/strategy", h.SetStrategy)
		v1.GET("/strategy", h.GetStrategy)
		v1.GET("/performance", h.GetPerformance)
	}
}

func (h *MarketMakingHandler) SetStrategy(c *gin.Context) {
	var cmd application.SetStrategyCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.app.SetStrategy(c.Request.Context(), cmd)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"strategy_id": id})
}

func (h *MarketMakingHandler) GetStrategy(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "symbol is required"})
		return
	}

	dto, err := h.app.GetStrategy(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if dto == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "strategy not found"})
		return
	}
	c.JSON(http.StatusOK, dto)
}

func (h *MarketMakingHandler) GetPerformance(c *gin.Context) {
	symbol := c.Query("symbol")
	dto, err := h.app.GetPerformance(c.Request.Context(), symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, dto)
}
