package http

import (
	"net/http"
	"time"

	"github.com/wyfcoding/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/wyfcoding/financialtrading/internal/pricing/application"
	"github.com/wyfcoding/financialtrading/internal/pricing/domain"
	"github.com/wyfcoding/pkg/logging"
)

// HTTP 处理器
// 负责处理与定价相关的 HTTP 请求
type PricingHandler struct {
	cmd   *application.PricingCommandService
	query *application.PricingQueryService
}

// 创建 HTTP 处理器实例
func NewPricingHandler(cmd *application.PricingCommandService, query *application.PricingQueryService) *PricingHandler {
	return &PricingHandler{cmd: cmd, query: query}
}

// 注册路由
// 将处理器方法绑定到 Gin 路由引擎
func (h *PricingHandler) RegisterRoutes(router *gin.RouterGroup) {
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
	DividendYield   float64               `json:"dividend_yield"`
	PricingModel    string                `json:"pricing_model"`
}

// GetOptionPrice 获取期权价格
func (h *PricingHandler) GetOptionPrice(c *gin.Context) {
	var req PricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	cmd := application.PriceOptionCommand{
		Symbol:          req.Contract.Symbol,
		OptionType:      req.Contract.Type,
		StrikePrice:     req.Contract.StrikePrice,
		ExpiryDate:      req.Contract.ExpiryDate.UnixMilli(),
		UnderlyingPrice: req.UnderlyingPrice,
		Volatility:      req.Volatility,
		RiskFreeRate:    req.RiskFreeRate,
		DividendYield:   req.DividendYield,
		PricingModel:    req.PricingModel,
	}

	result, err := h.cmd.PriceOption(c.Request.Context(), cmd)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to calculate option price", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{
		"price":            result.OptionPrice,
		"calculation_time": time.Now(),
	})
}

// GetGreeks 获取希腊字母
func (h *PricingHandler) GetGreeks(c *gin.Context) {
	var req PricingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ErrorWithStatus(c, http.StatusBadRequest, err.Error(), "")
		return
	}

	contract := domain.OptionContract{
		Symbol:      req.Contract.Symbol,
		Type:        domain.OptionType(req.Contract.Type),
		StrikePrice: decimal.NewFromFloat(req.Contract.StrikePrice),
		ExpiryDate:  req.Contract.ExpiryDate.UnixMilli(),
	}

	greeks, err := h.query.GetGreeks(c.Request.Context(), contract, decimal.NewFromFloat(req.UnderlyingPrice), req.Volatility, req.RiskFreeRate)
	if err != nil {
		logging.Error(c.Request.Context(), "Failed to calculate Greeks", "error", err)
		response.ErrorWithStatus(c, http.StatusInternalServerError, err.Error(), "")
		return
	}

	response.Success(c, gin.H{
		"greeks":           greeks,
		"calculation_time": time.Now(),
	})
}
