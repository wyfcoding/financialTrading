package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/wyfcoding/financialTrading/internal/pricing/application"
	"github.com/wyfcoding/financialTrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与定价相关的 HTTP 请求
type PricingHandler struct {
	app *application.PricingService // 定价应用服务
}

// 创建 HTTP 处理器实例
// app: 注入的定价应用服务
func NewPricingHandler(app *application.PricingService) *PricingHandler {
	return &PricingHandler{app: app}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *PricingHandler) RegisterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/pricing")
	{
		api.POST("/option/price", h.GetOptionPrice)
		api.POST("/option/greeks", h.GetGreeks)
	}
}

// OptionContractRequest 期权合约请求
type OptionContractRequest struct {
	Symbol      string    `json:"symbol" binding:"required"`
	Type        string    `json:"type" binding:"required"`
	StrikePrice float64   `json:"strike_price" binding:"required"`
	ExpiryDate  time.Time `json:"expiry_date" binding:"required"`
}

// PricingRequest 定价请求
type PricingRequest struct {
	Contract        OptionContractRequest `json:"contract" binding:"required"`
	UnderlyingPrice float64               `json:"underlying_price" binding:"required"`
	Volatility      float64               `json:"volatility" binding:"required"`
	RiskFreeRate    float64               `json:"risk_free_rate" binding:"required"`
}

// GetOptionPrice 获取期权价格
func (h *PricingHandler) GetOptionPrice(c *gin.Context) {
	var req PricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contract := domain.OptionContract{
		Symbol:      req.Contract.Symbol,
		Type:        domain.OptionType(req.Contract.Type),
		StrikePrice: req.Contract.StrikePrice,
		ExpiryDate:  req.Contract.ExpiryDate,
	}

	price, err := h.app.GetOptionPrice(c.Request.Context(), contract, req.UnderlyingPrice, req.Volatility, req.RiskFreeRate)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to calculate option price", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"price":            price,
		"calculation_time": time.Now(),
	})
}

// GetGreeks 获取希腊字母
func (h *PricingHandler) GetGreeks(c *gin.Context) {
	var req PricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	contract := domain.OptionContract{
		Symbol:      req.Contract.Symbol,
		Type:        domain.OptionType(req.Contract.Type),
		StrikePrice: req.Contract.StrikePrice,
		ExpiryDate:  req.Contract.ExpiryDate,
	}

	greeks, err := h.app.GetGreeks(c.Request.Context(), contract, req.UnderlyingPrice, req.Volatility, req.RiskFreeRate)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to calculate Greeks", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"greeks":           greeks,
		"calculation_time": time.Now(),
	})
}
